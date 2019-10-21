package main

import (
	"fmt"
	"github.com/adelolmo/hd-idle/diskstats"
	"github.com/adelolmo/hd-idle/io"
	"github.com/adelolmo/hd-idle/sgio"
	"log"
	"os"
	"time"
)

const (
	SCSI       = "scsi"
	ATA        = "ata"
	dateFormat = "2006-01-02T15:04:05"
)

type DefaultConf struct {
	Idle          int
	CommandType   string
	Debug         bool
	LogFile       string
	SymlinkPolicy int
}

type DeviceConf struct {
	Name        string
	GivenName   string
	Idle        int
	CommandType string
}

type Config struct {
	Devices  []DeviceConf
	Defaults DefaultConf
	SkewTime uint64
}

var previousSnapshots []diskstats.DiskStats
var now = time.Now()
var lastNow = time.Now()

func ObserveDiskActivity(config *Config) {
	actualSnapshot := diskstats.Snapshot()

	now = time.Now()
	resolveSymlinks(config)
	for _, stats := range actualSnapshot {
		updateState(stats, config)
	}
	lastNow = now
}

func resolveSymlinks(config *Config) {
	if config.Defaults.SymlinkPolicy == 0 {
		return
	}
	for i := range config.Devices {
		device := config.Devices[i]
		if len(device.Name) == 0 {
			realPath, err := io.RealPath(device.GivenName)
			if err == nil {
				config.Devices[i].Name = realPath
				logToFile(config.Defaults.LogFile,
					fmt.Sprintf("symlink %s resolved to %s", device.GivenName, realPath))
			}
			if err != nil && config.Defaults.Debug {
				fmt.Printf("Cannot resolve sysmlink %s\n", device.GivenName)
			}
		}
	}
}

func updateState(tmp diskstats.DiskStats, config *Config) {
	dsi := previousDiskStatsIndex(tmp.Name)
	if dsi < 0 {
		previousSnapshots = append(previousSnapshots, initDevice(tmp, config))
		return
	}

	if uint64(now.Sub(lastNow).Seconds()) > config.SkewTime {
		/* we slept too long, assume a suspend event and disks may be spun up */
		/* reset spin status and timers */
		previousSnapshots[dsi].SpinUpAt = now
		previousSnapshots[dsi].LastIoAt = now
		previousSnapshots[dsi].SpunDown = false
		logSpinupAfterSleep(previousSnapshots[dsi].Name, config.Defaults.LogFile)
	}

	ds := previousSnapshots[dsi]
	if ds.Writes == tmp.Writes && ds.Reads == tmp.Reads {
		if !ds.SpunDown {
			/* no activity on this disk and still running */
			idleDuration := int(now.Sub(ds.LastIoAt).Seconds())
			if ds.IdleTime != 0 && idleDuration > ds.IdleTime {
				spindownDisk(ds.Name, ds.CommandType)
				previousSnapshots[dsi].SpinDownAt = now
				previousSnapshots[dsi].SpunDown = true
			}
		}

	} else {
		/* disk had some activity */
		if ds.SpunDown {
			/* disk was spun down, thus it has just spun up */
			fmt.Printf("%s spinup\n", ds.Name)
			logSpinup(ds, config.Defaults.LogFile)
			previousSnapshots[dsi].SpinUpAt = now
		}
		previousSnapshots[dsi].Reads = tmp.Reads
		previousSnapshots[dsi].Writes = tmp.Writes
		previousSnapshots[dsi].LastIoAt = now
		previousSnapshots[dsi].SpunDown = false
	}

	if config.Defaults.Debug {
		ds = previousSnapshots[dsi]
		idleDuration := int(now.Sub(ds.LastIoAt).Seconds())
		fmt.Printf("disk=%s command=%s spunDown=%t "+
			"reads=%d writes=%d idleTime=%d idleDuration=%d "+
			"spindown=%s spinup=%s lastIO=%s\n",
			ds.Name, ds.CommandType, ds.SpunDown,
			ds.Reads, ds.Writes, ds.IdleTime, idleDuration,
			ds.SpinDownAt.Format(dateFormat), ds.SpinUpAt.Format(dateFormat), ds.LastIoAt.Format(dateFormat))
	}
}

func previousDiskStatsIndex(diskName string) int {
	for i, stats := range previousSnapshots {
		if stats.Name == diskName {
			return i
		}
	}
	return -1
}

func initDevice(stats diskstats.DiskStats, config *Config) diskstats.DiskStats {
	idle := config.Defaults.Idle
	command := config.Defaults.CommandType
	deviceConf := deviceConfig(stats.Name, config)
	if deviceConf != nil {
		idle = deviceConf.Idle
		command = deviceConf.CommandType
	}

	return diskstats.DiskStats{
		Name:        stats.Name,
		LastIoAt:    time.Now(),
		SpinUpAt:    time.Now(),
		SpunDown:    false,
		Writes:      stats.Writes,
		Reads:       stats.Reads,
		IdleTime:    idle,
		CommandType: command,
	}
}

func deviceConfig(diskName string, config *Config) *DeviceConf {
	for _, device := range config.Devices {
		if device.Name == diskName {
			return &device
		}
	}
	return &DeviceConf{
		Name:        diskName,
		CommandType: config.Defaults.CommandType,
		Idle:        config.Defaults.Idle,
	}
}

func spindownDisk(deviceName, command string) {
	fmt.Printf("%s spindown\n", deviceName)
	device := fmt.Sprintf("/dev/%s", deviceName)
	switch command {
	case SCSI:
		if err := sgio.StopScsiDevice(device); err != nil {
			fmt.Printf("cannot spindown scsi disk %s\n. %s", device, err.Error())
		}
		return
	case ATA:
		if err := sgio.StopAtaDevice(device); err != nil {
			fmt.Printf("cannot spindown ata disk %s\n. %s", device, err.Error())
		}
		return
	}
}

func logSpinup(ds diskstats.DiskStats, file string) {
	now := time.Now()
	text := fmt.Sprintf("date: %s, time: %s, disk: %s, running: %d, stopped: %d",
		now.Format("2006-01-02"), now.Format("15:04:05"), ds.Name,
		int(ds.SpinDownAt.Sub(ds.SpinUpAt).Seconds()), int(now.Sub(ds.SpinDownAt).Seconds()))
	logToFile(file, text)
}

func logSpinupAfterSleep(name string, file string) {
	text := fmt.Sprintf("date: %s, time: %s, disk: %s, assuming disk spun up after long sleep",
		now.Format("2006-01-02"), now.Format("15:04:05"), name)
	logToFile(file, text)
}

func logToFile(file string, text string) {
	if len(file) > 0 {
		cacheFile, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatalf("Cannot open file %s. Error: %s", file, err)
		}
		if _, err = cacheFile.WriteString(text + "\n"); err != nil {
			log.Fatalf("Cannot write into file %s. Error: %s", file, err)
		}
		err = cacheFile.Close()
		if err != nil {
			log.Fatalf("Cannot close file %s. Error: %s", file, err)
		}
	}
}

func (c *Config) String() string {
	return fmt.Sprintf("symlinkPolicy=%d, defaultIdle=%d, defaultCommand=%s, debug=%t, devices=%v, logFile=%s",
		c.Defaults.SymlinkPolicy, c.Defaults.Idle, c.Defaults.CommandType, c.Defaults.Debug, c.Devices, c.Defaults.LogFile)
}
