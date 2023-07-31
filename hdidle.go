// hd-idle - spin down idle hard disks
// Copyright (C) 2018  Andoni del Olmo
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"github.com/adelolmo/hd-idle/diskstats"
	"github.com/adelolmo/hd-idle/io"
	"github.com/adelolmo/hd-idle/sgio"
	"log"
	"math"
	"os"
	"time"
)

const (
	SCSI       = "scsi"
	ATA        = "ata"
	dateFormat = "2006-01-02T15:04:05"
)

type DefaultConf struct {
	Idle           time.Duration
	CommandType    string
	PowerCondition uint8
	Debug          bool
	LogFile        string
	SymlinkPolicy  int
}

type DeviceConf struct {
	Name           string
	GivenName      string
	Idle           time.Duration
	CommandType    string
	PowerCondition uint8
}

type Config struct {
	Devices  []DeviceConf
	Defaults DefaultConf
	SkewTime time.Duration
	NameMap  map[string]string
}

func (c *Config) resolveDeviceGivenName(name string) string {
	if givenName, ok := c.NameMap[name]; ok {
		return givenName
	}
	return name
}

type DiskStats struct {
	Name           string
	GivenName      string
	IdleTime       time.Duration
	CommandType    string
	PowerCondition uint8
	Reads          uint64
	Writes         uint64
	SpinDownAt     time.Time
	SpinUpAt       time.Time
	LastIoAt       time.Time
	SpunDown       bool
}

var previousSnapshots []DiskStats
var now = time.Now()
var lastNow = time.Now()

func ObserveDiskActivity(config *Config) {
	actualSnapshot := diskstats.Snapshot()

	now = time.Now()
	resolveSymlinks(config)
	for _, stats := range actualSnapshot {
		d := &DiskStats{
			Name:   stats.Name,
			Reads:  stats.Reads,
			Writes: stats.Writes,
		}
		updateState(*d, config)
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

func updateState(tmp DiskStats, config *Config) {
	dsi := previousDiskStatsIndex(tmp.Name)
	if dsi < 0 {
		previousSnapshots = append(previousSnapshots, initDevice(tmp, config))
		return
	}

	intervalDurationInSeconds := now.Unix() - lastNow.Unix()
	if intervalDurationInSeconds > config.SkewTime.Milliseconds()/1000 {
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
			idleDuration := now.Sub(ds.LastIoAt)
			if ds.IdleTime != 0 && idleDuration > ds.IdleTime {
				log.Printf("%s spindown\n", config.resolveDeviceGivenName(ds.Name))
				device := fmt.Sprintf("/dev/%s", ds.Name)
				if err := spindownDisk(device, ds.CommandType, ds.PowerCondition); err != nil {
					fmt.Println(err.Error())
				}
				previousSnapshots[dsi].SpinDownAt = now
				previousSnapshots[dsi].SpunDown = true
			}
		}

	} else {
		/* disk had some activity */
		if ds.SpunDown {
			/* disk was spun down, thus it has just spun up */
			log.Printf("%s spinup\n", config.resolveDeviceGivenName(ds.Name))
			logSpinup(ds, config.Defaults.LogFile, config.resolveDeviceGivenName(ds.Name))
			previousSnapshots[dsi].SpinUpAt = now
		}
		previousSnapshots[dsi].Reads = tmp.Reads
		previousSnapshots[dsi].Writes = tmp.Writes
		previousSnapshots[dsi].LastIoAt = now
		previousSnapshots[dsi].SpunDown = false
	}

	if config.Defaults.Debug {
		ds = previousSnapshots[dsi]
		idleDuration := now.Sub(ds.LastIoAt)
		fmt.Printf("disk=%s command=%s spunDown=%t "+
			"reads=%d writes=%d idleTime=%v idleDuration=%v "+
			"spindown=%s spinup=%s lastIO=%s\n",
			ds.Name, ds.CommandType, ds.SpunDown,
			ds.Reads, ds.Writes, ds.IdleTime.Seconds(), math.RoundToEven(idleDuration.Seconds()),
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

func initDevice(stats DiskStats, config *Config) DiskStats {
	idle := config.Defaults.Idle
	command := config.Defaults.CommandType
	powerCondition := config.Defaults.PowerCondition
	deviceConf := deviceConfig(stats.Name, config)
	if deviceConf != nil {
		idle = deviceConf.Idle
		command = deviceConf.CommandType
		powerCondition = deviceConf.PowerCondition
	}

	return DiskStats{
		Name:           stats.Name,
		LastIoAt:       time.Now(),
		SpinUpAt:       time.Now(),
		SpunDown:       false,
		Writes:         stats.Writes,
		Reads:          stats.Reads,
		IdleTime:       idle,
		CommandType:    command,
		PowerCondition: powerCondition,
	}
}

func deviceConfig(diskName string, config *Config) *DeviceConf {
	for _, device := range config.Devices {
		if device.Name == diskName {
			return &device
		}
	}
	return &DeviceConf{
		Name:           diskName,
		CommandType:    config.Defaults.CommandType,
		PowerCondition: config.Defaults.PowerCondition,
		Idle:           config.Defaults.Idle,
	}
}

func spindownDisk(device, command string, powerCondition uint8) error {
	switch command {
	case SCSI:
		if err := sgio.StartStopScsiDevice(device, powerCondition); err != nil {
			return fmt.Errorf("cannot spindown scsi disk %s:\n%s\n", device, err.Error())
		}
		return nil
	case ATA:
		if err := sgio.StopAtaDevice(device); err != nil {
			return fmt.Errorf("cannot spindown ata disk %s:\n%s\n", device, err.Error())
		}
		return nil
	}
	return nil
}

func logSpinup(ds DiskStats, file, givenName string) {
	now := time.Now()
	text := fmt.Sprintf("date: %s, time: %s, disk: %s, running: %d, stopped: %d",
		now.Format("2006-01-02"), now.Format("15:04:05"), givenName,
		int(ds.SpinDownAt.Sub(ds.SpinUpAt).Seconds()), int(now.Sub(ds.SpinDownAt).Seconds()))
	logToFile(file, text)
}

func logSpinupAfterSleep(name, file string) {
	text := fmt.Sprintf("date: %s, time: %s, disk: %s, assuming disk spun up after long sleep",
		now.Format("2006-01-02"), now.Format("15:04:05"), name)
	logToFile(file, text)
}

func logToFile(file, text string) {
	if len(file) == 0 {
		return
	}

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

func (c *Config) String() string {
	var devices string
	for _, device := range c.Devices {
		devices += "{" + device.String() + "}"
	}
	return fmt.Sprintf("symlinkPolicy=%d, defaultIdle=%v, defaultCommand=%s, defaultPowerCondition=%v, debug=%t, logFile=%s, devices=%s",
		c.Defaults.SymlinkPolicy, c.Defaults.Idle.Seconds(), c.Defaults.CommandType, c.Defaults.PowerCondition, c.Defaults.Debug, c.Defaults.LogFile, devices)
}

func (dc *DeviceConf) String() string {
	return fmt.Sprintf("name=%s, givenName=%s, idle=%v, commandType=%s, powerCondition=%v",
		dc.Name, dc.GivenName, dc.Idle.Seconds(), dc.CommandType, dc.PowerCondition)
}
