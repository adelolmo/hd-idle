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
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	SCSI       = "scsi"
	ATA        = "ata"
	dateFormat = "2006-01-02T15:04:05"
)

type DefaultConf struct {
	Idle                    time.Duration
	CommandType             string
	PowerCondition          uint8
	Debug                   bool
	Verbose                 bool // New field for verbose output
	LogFile                 string
	SymlinkPolicy           int
	IgnoreSpinDownDetection bool
	SkipIfMounted           bool
	DryRun                  bool
}

type DeviceConf struct {
	Name           string
	GivenName      string
	Idle           time.Duration
	CommandType    string
	PowerCondition uint8
	SkipIfMounted  bool
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
	LastSpunDownAt time.Time
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
			if err != nil && config.Defaults.Verbose {
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

	// Ensure status directory exists
	statusDir := "/var/run/hd-idle"
	if _, err := os.Stat(statusDir); os.IsNotExist(err) {
		os.MkdirAll(statusDir, 0755)
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
		if !ds.SpunDown || config.Defaults.IgnoreSpinDownDetection {

			idleDuration := now.Sub(ds.LastIoAt)
			timeSinceLastSpunDown := now.Sub(ds.LastSpunDownAt)

			if ds.IdleTime != 0 && idleDuration > ds.IdleTime && timeSinceLastSpunDown > ds.IdleTime {
				// Check if spindown should be skipped if the device is mounted
				skip := config.Defaults.SkipIfMounted
				if deviceConf := deviceConfig(ds.Name, config); deviceConf != nil {
					skip = deviceConf.SkipIfMounted
				}

				if skip {
					devicePath := fmt.Sprintf("/dev/%s", ds.Name)
					mounted, err := isDeviceMounted(devicePath, config.Defaults.Debug)
					if err != nil {
						fmt.Printf("Error checking mount status for %s: %v\n", config.resolveDeviceGivenName(ds.Name), err)
					}
					if mounted {
						if config.Defaults.Verbose {
							// Show debug info for Btrfs/ZFS
							if isBtrfs, _ := CheckBtrfsDevice(devicePath); isBtrfs {
								fmt.Printf("Spindown skipped: %s is part of a mounted Btrfs filesystem\n", devicePath)
							} else if isZfs, _ := CheckZfsDevice(devicePath); isZfs {
								fmt.Printf("Spindown skipped: %s is part of a mounted ZFS filesystem\n", devicePath)
							}
						}
						return
					}
				}

				if config.Defaults.Debug || config.Defaults.Verbose || len(config.Defaults.LogFile) > 0 {
					if ds.SpunDown && config.Defaults.IgnoreSpinDownDetection {
						fmt.Printf("%s spindown (ignoring prior spin down state)\n",
							config.resolveDeviceGivenName(ds.Name))
					} else {
						fmt.Printf("%s spindown\n",
							config.resolveDeviceGivenName(ds.Name))
					}
				}
				device := fmt.Sprintf("/dev/%s", ds.Name)
				if err := spindownDisk(device, ds.CommandType, ds.PowerCondition, config.Defaults.Debug, config.Defaults.DryRun); err != nil {
					fmt.Println(err.Error())
				}
				previousSnapshots[dsi].LastSpunDownAt = now
				previousSnapshots[dsi].SpinDownAt = now
				previousSnapshots[dsi].SpunDown = true

				// Create sdX.idle file
				idleFile := fmt.Sprintf("/var/run/hd-idle/%s.idle", ds.Name)
				if err := os.WriteFile(idleFile, []byte{}, 0644); err != nil && config.Defaults.Debug {
					fmt.Printf("Failed to write idle file: %v\n", err)
				}
			}
		}

	} else {
		/* disk had some activity */
		if ds.SpunDown {
			/* disk was spun down, thus it has just spun up */
			if config.Defaults.Debug || config.Defaults.Verbose || len(config.Defaults.LogFile) > 0 {
				fmt.Printf("%s spinup\n", config.resolveDeviceGivenName(ds.Name))
			}
			logSpinup(ds, config.Defaults.LogFile, config.resolveDeviceGivenName(ds.Name))
			previousSnapshots[dsi].SpinUpAt = now

			// Remove sdX.idle file
			idleFile := fmt.Sprintf("/var/run/hd-idle/%s.idle", ds.Name)
			if err := os.Remove(idleFile); err != nil && config.Defaults.Debug {
				fmt.Printf("Failed to remove idle file: %v\n", err)
			}
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
			"spindown=%s spinup=%s lastIO=%s \n",
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
		SkipIfMounted:  config.Defaults.SkipIfMounted,
	}
}

func spindownDisk(device, command string, powerCondition uint8, debug, dryRun bool) error {
	if dryRun {
		if debug {
			fmt.Printf("Dry-run: Would spindown %s (command: %s, powerCondition: %d)\n", device, command, powerCondition)
		}
		return nil
	}

	switch command {
	case SCSI:
		if err := sgio.StartStopScsiDevice(device, powerCondition); err != nil {
			return fmt.Errorf("cannot spindown scsi disk %s:\n%s\n", device, err.Error())
		}
		return nil
	case ATA:
		if err := sgio.StopAtaDevice(device, debug); err != nil {
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
	return fmt.Sprintf("symlinkPolicy=%d, defaultIdle=%v, defaultCommand=%s, defaultPowerCondition=%v, debug=%t, verbose=%t, logFile=%s, devices=%s, ignoreSpinDownDetection=%t, skipIfMounted=%t",
		c.Defaults.SymlinkPolicy, c.Defaults.Idle.Seconds(), c.Defaults.CommandType, c.Defaults.PowerCondition, c.Defaults.Debug, c.Defaults.Verbose, c.Defaults.LogFile, devices, c.Defaults.IgnoreSpinDownDetection, c.Defaults.SkipIfMounted)
}

func (dc *DeviceConf) String() string {
	return fmt.Sprintf("name=%s, givenName=%s, idle=%v, commandType=%s, powerCondition=%v, skipIfMounted=%t",
		dc.Name, dc.GivenName, dc.Idle.Seconds(), dc.CommandType, dc.PowerCondition, dc.SkipIfMounted)
}

// isDeviceMounted checks if a given device is currently mounted, including Btrfs and ZFS filesystems.
func isDeviceMounted(device string, verbose bool) (bool, error) {
	// Check if device is part of a mounted btrfs or zfs filesystem
	if isBtrfs, _ := CheckBtrfsDevice(device); isBtrfs {
		if verbose {
			fmt.Printf("%s is part of a mounted Btrfs filesystem\n", device)
		}
		return true, nil
	}
	if isZfs, _ := CheckZfsDevice(device); isZfs {
		if verbose {
			fmt.Printf("%s is part of a mounted ZFS filesystem\n", device)
		}
		return true, nil
	}

	// Fallback to generic findmnt check if not Btrfs or ZFS
	cmd := exec.Command("findmnt", "-n", "-o", "TARGET", device)
	output, err := cmd.Output()
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		if verbose {
			fmt.Printf("%s is mounted directly (generic check)\n", device)
		}
		return true, nil
	}

	// If the device itself is not mounted, check its partitions
	partitions := GetDevicePartitions(device)
	for _, part := range partitions {
		cmd = exec.Command("findmnt", "-n", "-o", "TARGET", part)
		output, err = cmd.Output()
		if err == nil && len(strings.TrimSpace(string(output))) > 0 {
			if verbose {
				fmt.Printf("%s (partition of %s) is mounted (generic check)\n", part, device)
			}
			return true, nil
		}
	}

	if verbose {
		fmt.Printf("%s and its partitions are not mounted\n", device)
	}
	return false, nil
}

// GetDevicePartitions lists all partitions for a given device.
func GetDevicePartitions(device string) []string {
	var partitions []string
	cmd := exec.Command("lsblk", "-ln", "-o", "NAME", device)
	output, err := cmd.Output()
	if err != nil {
		return partitions
	}

	deviceName := filepath.Base(device)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, deviceName) && len(line) > len(deviceName) {
			partitions = append(partitions, "/dev/"+line)
		}
	}
	return partitions
}

// CheckBtrfsDevice checks if a device is part of a mounted Btrfs filesystem
func CheckBtrfsDevice(device string) (bool, string) {
	// Check if device is valid
	if !FileExists(device) {
		return false, fmt.Sprintf("Error: %s is not a valid block device.", device)
	}

	// Get mounted btrfs filesystems
	mounts, err := exec.Command("mount", "-t", "btrfs").Output()
	if err != nil {
		return false, fmt.Sprintf("%s is not part of any mounted Btrfs filesystem.", device)
	}

	// Parse mount points
	mountPoints := ParseMountPoints(string(mounts))
	if len(mountPoints) == 0 {
		return false, fmt.Sprintf("%s is not part of any mounted Btrfs filesystem.", device)
	}

	// Check each mount point
	for _, mountPoint := range mountPoints {
		// Run btrfs filesystem show for this mount point
		cmd := exec.Command("btrfs", "filesystem", "show", mountPoint)
		output, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}

		// Check if device is in the output
		if strings.Contains(string(output), device) {
			// Check if device is marked as missing
			if strings.Contains(string(output), device+" missing") {
				return true, fmt.Sprintf("%s is part of a Btrfs filesystem at %s but is marked as MISSING.", device, mountPoint)
			}
			return true, fmt.Sprintf("%s is part of a mounted Btrfs filesystem at %s.\nStatus: %s is active in the Btrfs filesystem.", device, mountPoint, device)
		}
	}

	return false, fmt.Sprintf("%s is not part of any mounted Btrfs filesystem.", device)
}

// CheckZfsDevice checks if a device is part of a ZFS pool
func CheckZfsDevice(device string) (bool, string) {
	// Check if device is valid
	if !FileExists(device) {
		return false, fmt.Sprintf("Error: %s is not a valid block device.", device)
	}

	// Get device serial
	devID := GetDeviceSerial(device)

	// Get partitions
	partitions := GetDevicePartitions(device)

	// Get ZFS pools
	pools, err := exec.Command("zpool", "list", "-H", "-o", "name").Output()
	if err != nil {
		return false, fmt.Sprintf("%s is not part of any ZFS pool.", device)
	}

	// Parse pool names
	poolNames := strings.Fields(string(pools))
	if len(poolNames) == 0 {
		return false, fmt.Sprintf("%s is not part of any ZFS pool.", device)
	}

	// Check each pool
	for _, pool := range poolNames {
		// Build pattern for grep
		pattern := device
		if devID != "" {
			pattern += "|" + devID
		}
		for _, part := range partitions {
			pattern += "|" + part
		}

		// Check if device is in zpool status
		cmd := exec.Command("zpool", "status", pool)
		output, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}

		// Check if device matches pattern
		matched, _ := regexp.MatchString(pattern, string(output))
		if matched {
			// Check if pool is mounted
			mountOutput, _ := exec.Command("mount").Output()
			if strings.Contains(string(mountOutput), "zfs") && strings.Contains(string(mountOutput), pool) {
				// Get mount point
				lines := strings.Split(string(mountOutput), "\n")
				var mpFound string
				for _, line := range lines {
					if strings.Contains(line, "zfs") && strings.Contains(line, pool) {
						fields := strings.Fields(line)
						if len(fields) > 2 {
							mpFound = fields[2]
							break
						}
					}
				}

				// Get device state
				lines = strings.Split(string(output), "\n")
				var state string
				for _, line := range lines {
					if regexp.MustCompile(pattern).MatchString(line) {
						fields := strings.Fields(line)
						if len(fields) > 1 {
							state = fields[1]
							break
						}
					}
				}

				if state == "ONLINE" {
					return true, fmt.Sprintf("%s is part of a mounted ZFS filesystem in pool %s at %s.\nStatus: %s is active in the ZFS pool.", device, pool, mpFound, device)
				}
				return true, fmt.Sprintf("%s is part of a ZFS pool %s but is in state %s.", device, pool, state)
			}
			return true, fmt.Sprintf("%s is part of ZFS pool %s but the pool is not mounted.", device, pool)
		}
	}

	return false, fmt.Sprintf("%s is not part of any ZFS pool.", device)
}

// Helper functions
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func ParseMountPoints(mounts string) []string {
	var mountPoints []string
	lines := strings.Split(mounts, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			mountPoints = append(mountPoints, fields[2])
		}
	}
	return mountPoints
}

func GetDeviceSerial(device string) string {
	cmd := exec.Command("lsblk", "-no", "SERIAL", device)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
