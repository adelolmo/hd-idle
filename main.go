package main

import (
	"errors"
	"fmt"
	"github.com/adelolmo/hd-idle/device"
	"github.com/adelolmo/hd-idle/sgio"
	"github.com/jasonlvhit/gocron"
	"os"
	"strconv"
)

const defaultIdleTime = 600

func main() {

	if os.Getenv("START_HD_IDLE") == "false" {
		println("START_HD_IDLE=false exiting now.")
		os.Exit(0)
	}

	defaultConf := DefaultConf{
		Idle:        defaultIdleTime,
		CommandType: SCSI,
		Debug:       false,
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
				println("Missing disk argument. Must be a device (e.g. sda)")
				os.Exit(1)
			}
			disk := os.Args[index+2]
			sgio.StopScsiDevice(disk)
			os.Exit(0)

		case "-a":
			if deviceConf != nil {
				config.Devices = append(config.Devices, *deviceConf)
			}

			name := os.Args[index+2]
			deviceConf = &DeviceConf{
				Name:        device.RealPath(name),
				Idle:        config.Defaults.Idle,
				CommandType: config.Defaults.CommandType,
			}

		case "-i":
			s := os.Args[index+2]
			idle, err := strconv.Atoi(s)
			if err != nil {
				println(errors.New(fmt.Sprintf("Wrong idle_time -i %d. Must be a number", idle)))
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
				println(errors.New(fmt.Sprintf("Wrong command_type -c %s. Must be one of: scsi, ata", command)))
				os.Exit(1)
			}

		case "-l":
			config.Defaults.LogFile = os.Args[index+2]

		case "-d":
			config.Defaults.Debug = true

		case "h":
			println("usage: hd-idle [-t <disk.go>] [-a <name>] [-i <idle_time>] [-c <command_type>] [-l <logfile>] [-d] [-h]\n")
			os.Exit(0)
		}
	}

	if deviceConf != nil {
		config.Devices = append(config.Devices, *deviceConf)
	}
	println(config.String())

	interval := poolInterval(config.Devices)
	gocron.Every(interval).Seconds().Do(ObserveDiskActivity, config)
	gocron.NextRun()
	<-gocron.Start()
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
