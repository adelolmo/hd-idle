package main

import (
	"fmt"
	"github.com/adelolmo/hd-idle/io"
	"github.com/adelolmo/hd-idle/sgio"
	"github.com/go-co-op/gocron"
	"os"
	"strconv"
	"time"
)

const (
	defaultIdleTime     = 600
	symlinkResolveOnce  = 0
	symlinkResolveRetry = 1
)

func main() {

	if os.Getenv("START_HD_IDLE") == "false" {
		fmt.Println("START_HD_IDLE=false exiting now.")
		os.Exit(0)
	}

	defaultConf := DefaultConf{
		Idle:          defaultIdleTime,
		CommandType:   SCSI,
		Debug:         false,
		SymlinkPolicy: 0,
	}
	var config = &Config{
		Devices:  []DeviceConf{},
		Defaults: defaultConf,
	}
	var deviceConf *DeviceConf

	for index, arg := range os.Args[1:] {
		switch arg {
		case "-t":
			if len(os.Args) < 3 {
				fmt.Println("Missing disk argument. Must be a device (e.g. sda)")
				os.Exit(1)
			}
			disk := os.Args[index+2]
			if err := sgio.StopScsiDevice(disk); err != nil {
				fmt.Printf("cannot spindown scsi disk %s\n. %s", disk, err.Error())
				os.Exit(1)
			}
			os.Exit(0)

		case "-s":
			s := os.Args[index+2]
			switch s {
			case "0":
				config.Defaults.SymlinkPolicy = symlinkResolveOnce
			case "1":
				config.Defaults.SymlinkPolicy = symlinkResolveRetry
			default:
				fmt.Printf("Wrong symlink_policy -s %s. Must be 0 or 1\n", s)
				os.Exit(1)
			}

		case "-a":
			if deviceConf != nil {
				config.Devices = append(config.Devices, *deviceConf)
			}

			name := os.Args[index+2]
			deviceRealPath, err := io.RealPath(name)
			if err != nil {
				deviceRealPath = ""
				fmt.Printf("Unable to resolve symlink: %s\n", name)
			}
			//println("name: " + deviceRealPath + " givenName: " + name)
			deviceConf = &DeviceConf{
				Name:        deviceRealPath,
				GivenName:   name,
				Idle:        config.Defaults.Idle,
				CommandType: config.Defaults.CommandType,
			}

		case "-i":
			s := os.Args[index+2]
			idle, err := strconv.Atoi(s)
			if err != nil {
				fmt.Printf("Wrong idle_time -i %d. Must be a number", idle)
				os.Exit(1)
			}
			if deviceConf == nil {
				config.Defaults.Idle = idle
				break
			}
			deviceConf.Idle = idle

		case "-c":
			command := os.Args[index+2]
			switch command {
			case SCSI, ATA:
				if deviceConf == nil {
					config.Defaults.CommandType = command
					break
				}
				deviceConf.CommandType = command
			default:
				fmt.Printf("Wrong command_type -c %s. Must be one of: scsi, ata", command)
				os.Exit(1)
			}

		case "-l":
			config.Defaults.LogFile = os.Args[index+2]

		case "-d":
			config.Defaults.Debug = true

		case "h":
			fmt.Println("usage: hd-idle [-t <disk>] [-s <symlink_policy>] [-a <name>] [-i <idle_time>] " +
				"[-c <command_type>] [-l <logfile>] [-d] [-h]")
			os.Exit(0)
		}
	}

	if deviceConf != nil {
		config.Devices = append(config.Devices, *deviceConf)
	}
	fmt.Println(config.String())

	interval := poolInterval(config.Devices)
	config.SkewTime = interval * 3
	s := gocron.NewScheduler(time.UTC)
	_, _ = s.Every(interval).Second().Do(ObserveDiskActivity, config)
	<- s.Start()
}

func poolInterval(deviceConfs []DeviceConf) uint64 {
	var interval = ^uint64(0)

	if len(deviceConfs) == 0 {
		return defaultIdleTime / 10
	}

	for _, dev := range deviceConfs {
		if uint64(dev.Idle) < interval {
			interval = uint64(dev.Idle)
		}
	}

	sleepTime := interval / 10
	if sleepTime == 0 {
		return 1
	}
	return sleepTime
}
