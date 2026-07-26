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
	"github.com/adelolmo/hd-idle/io"
	"os"
	"strconv"
	"time"
)

const (
	defaultIdleTime     = 600 * time.Second
	symlinkResolveOnce  = 0
	symlinkResolveRetry = 1
)

func main() {

	if os.Getenv("START_HD_IDLE") == "false" {
		fmt.Println("START_HD_IDLE=false exiting now.")
		os.Exit(0)
	}

	singleDiskMode := false
	testMode := false
	var disk string
	defaultConf := DefaultConf{
		Idle:           defaultIdleTime,
		CommandType:    SCSI,
		PowerCondition: 0,
		Debug:          false,
		Verbose:        false, // Initialize new verbose flag
		SymlinkPolicy:  0,
		SkipIfMounted:  false,
		DryRun:         false,
	}
	var config = &Config{
		Devices:  []DeviceConf{},
		Defaults: defaultConf,
		NameMap:  map[string]string{},
	}
	var deviceConf *DeviceConf

	if len(os.Args) == 0 {
		usage()
		os.Exit(1)
	}

	for index, arg := range os.Args[1:] {
		switch arg {
		case "-t":
			var err error
			disk, err = argument(index)
			if err != nil {
				fmt.Println("Missing disk argument after -t. Must be a device (e.g. -t sda).")
				os.Exit(1)
			}
			singleDiskMode = true

		case "-T":
			var err error
			disk, err = argument(index)
			if err != nil {
				fmt.Println("Missing disk argument after -T. Must be a device (e.g. -T sda).")
				os.Exit(1)
			}
			testMode = true

		case "-s":
			s, err := argument(index)
			if err != nil {
				fmt.Println("Missing symlink_policy. Must be 0 or 1.")
				os.Exit(1)
			}
			switch s {
			case "0":
				config.Defaults.SymlinkPolicy = symlinkResolveOnce
			case "1":
				config.Defaults.SymlinkPolicy = symlinkResolveRetry
			default:
				fmt.Printf("Wrong symlink_policy -s %s. Must be 0 or 1.\n", s)
				os.Exit(1)
			}

		case "-a":
			if deviceConf != nil {
				config.Devices = append(config.Devices, *deviceConf)
			}

			name, err := argument(index)
			if err != nil {
				fmt.Println("Missing disk argument after -a. Must be a device (e.g. -a sda).")
				os.Exit(1)
			}

			deviceRealPath, err := io.RealPath(name)
			if err != nil {
				deviceRealPath = ""
				fmt.Printf("Unable to resolve symlink: %s\n", name)
			}
			deviceConf = &DeviceConf{
				Name:           deviceRealPath,
				GivenName:      name,
				Idle:           config.Defaults.Idle,
				CommandType:    config.Defaults.CommandType,
				PowerCondition: config.Defaults.PowerCondition,
			}
			config.NameMap[deviceRealPath] = name
			deviceConf.SkipIfMounted = config.Defaults.SkipIfMounted

		case "-i":
			s, err := argument(index)
			if err != nil {
				fmt.Println("Missing idle_time after -i. Must be a number.")
				os.Exit(1)
			}
			idle, err := strconv.Atoi(s)
			if err != nil {
				fmt.Printf("Wrong idle_time -i %d. Must be a number.", idle)
				os.Exit(1)
			}
			if deviceConf == nil {
				config.Defaults.Idle = time.Duration(idle) * time.Second
				break
			}
			deviceConf.Idle = time.Duration(idle) * time.Second

		case "-I":
			config.Defaults.IgnoreSpinDownDetection = true

		case "-c":
			command, err := argument(index)
			if err != nil {
				fmt.Println("Missing command_type after -c. Must be one of: scsi, ata.")
				os.Exit(1)
			}
			switch command {
			case SCSI, ATA:
				if deviceConf == nil {
					config.Defaults.CommandType = command
					break
				}
				deviceConf.CommandType = command
			default:
				fmt.Printf("Wrong command_type -c %s. Must be one of: scsi, ata.", command)
				os.Exit(1)
			}

		case "-p":
			s, err := argument(index)
			if err != nil {
				fmt.Println("Missing power condition after -p. Must be a number from 0-15.")
				os.Exit(1)
			}
			powerCondition, err := strconv.ParseUint(s, 0, 4)
			if err != nil {
				fmt.Printf("Invalid power condition %s: %s", s, err.Error())
				os.Exit(1)
			}
			if deviceConf == nil {
				config.Defaults.PowerCondition = uint8(powerCondition)
				break
			}
			deviceConf.PowerCondition = uint8(powerCondition)

		case "-l":
			logfile, err := argument(index)
			if err != nil {
				fmt.Println("Missing logfile after -l.")
				os.Exit(1)
			}
			config.Defaults.LogFile = logfile

		case "-d":
			config.Defaults.Debug = true

		case "-v":
			config.Defaults.Verbose = true

		case "-M":
			config.Defaults.SkipIfMounted = true
			if deviceConf != nil {
				deviceConf.SkipIfMounted = true
			}

		case "-N":
			config.Defaults.DryRun = true

		case "-h":
			usage()
			os.Exit(0)
		}
	}

	if singleDiskMode {
		if err := spindownDisk(
			disk,
			config.Defaults.CommandType,
			config.Defaults.PowerCondition,
			config.Defaults.Debug,
			config.Defaults.DryRun,
		); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}

	if testMode {
		// Check if device is part of a mounted btrfs or zfs filesystem
		if isBtrfs, btrfsInfo := CheckBtrfsDevice(disk); isBtrfs {
			fmt.Printf("Spindown skipped: %s\n", btrfsInfo)
			os.Exit(0)
		}
		if isZfs, zfsInfo := CheckZfsDevice(disk); isZfs {
			fmt.Printf("Spindown skipped: %s\n", zfsInfo)
			os.Exit(0)
		}
		fmt.Printf("%s is not part of any mounted Btrfs or ZFS filesystem.\n", disk)
		os.Exit(0)
	}

	if deviceConf != nil {
		config.Devices = append(config.Devices, *deviceConf)
	}
	fmt.Println(config.String())

	interval := poolInterval(config.Devices)
	config.SkewTime = interval * 3
	for {
		ObserveDiskActivity(config)
		time.Sleep(interval)
	}
}

func argument(index int) (string, error) {
	argIndex := index + 2
	if argIndex >= len(os.Args) {
		return "", fmt.Errorf("option requires argument")
	}
	arg := os.Args[argIndex]
	if arg[:1] == "-" {
		return "", fmt.Errorf("option requires argument")
	}
	return arg, nil
}

func usage() {
	fmt.Println(`usage: hd-idle [options]

Options:
  -t <disk>             Test mode: spindown the specified disk immediately.
  -T <disk>             Test mode: check if the specified disk is part of a mounted Btrfs or ZFS filesystem and exit.
  -s <symlink_policy>   Symlink policy: 0 (resolve once), 1 (resolve on retry). Default is 0.
  -a <name>             Add a device to monitor. Can be a device name (e.g., sda) or a symlink (e.g., /dev/disk/by-id/...).
                        This option can be used multiple times for multiple devices.
  -i <idle_time>        Idle time in seconds before spindown. Applies to the last specified device (-a) or as a default.
  -c <command_type>     Command type for spindown: 'scsi' or 'ata'. Applies to the last specified device (-a) or as a default.
  -p <power_condition>  Power condition for SCSI devices (0-15). Applies to the last specified device (-a) or as a default.
  -l <logfile>          Path to a log file for recording spinup events.
  -d                    Enable debug output (disk status).
  -v                    Enable verbose output (filesystem checks).
  -I                    Ignore prior spindown state (force spindown even if already spun down).
  -M                    Skip spindown if the device is currently mounted. Applies to the last specified device (-a) or as a default.
  -N                    Dry-run mode: simulate spindown actions without executing them.
  -h                    Display this help message.`)
}

func poolInterval(deviceConfs []DeviceConf) time.Duration {
	if len(deviceConfs) == 0 {
		return defaultIdleTime / 10
	}

	interval := defaultIdleTime
	for _, dev := range deviceConfs {
		if dev.Idle == 0 {
			continue
		}
		if dev.Idle < interval {
			interval = dev.Idle
		}
	}

	sleepTime := interval / 10
	if sleepTime == 0 {
		return time.Second
	}
	return sleepTime
}
