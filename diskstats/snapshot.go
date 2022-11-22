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

package diskstats

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

/*
https://www.kernel.org/doc/Documentation/ABI/testing/procfs-diskstats

The /proc/diskstats file displays the I/O statistics
of block devices. Each line contains the following 14
fields:

1 - major number
2 - minor mumber
3 - device name
4 - reads completed successfully
5 - reads merged
6 - sectors read
7 - time spent reading (ms)
8 - writes completed
9 - writes merged
10 - sectors written
11 - time spent writing (ms)
12 - I/Os currently in progress
13 - time spent doing I/Os (ms)
14 - weighted time spent doing I/Os (ms)

Kernel 4.18+ appends four more fields for discard
tracking putting the total at 18:

15 - discards completed successfully
16 - discards merged
17 - sectors discarded
18 - time spent discarding

Kernel 5.5+ appends two more fields for flush requests:

19 - flush requests completed successfully
20 - time spent flushing

For more details refer to Documentation/admin-guide/iostats.rst
*/

const (
	deviceNameCol = 2 // field 3 - device name
	readsCol      = 5 // field 6 - sectors read
	writesCol     = 9 // field 10 - sectors written
)

type DeviceType int

const (
	Unknown DeviceType = iota
	Disk
	Partition
	DeviceMapper
)

type ReadWriteStats struct {
	Name   string
	Type   DeviceType
	Reads  uint64
	Writes uint64
}

var scsiDiskRegex *regexp.Regexp
var scsiPartitionRegex *regexp.Regexp
var deviceMapperRegex *regexp.Regexp

type diskHolderGetterFunc func(string, string) (string, error)

func init() {
	scsiDiskRegex = regexp.MustCompile("sd[a-z]+$")
	scsiPartitionRegex = regexp.MustCompile("sd[a-z]+[0-9]+$")
	deviceMapperRegex = regexp.MustCompile("dm-.*$")
}

func Snapshot() []ReadWriteStats {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	return readSnapshot(f, getDiskHolder)
}

func readSnapshot(r io.Reader, holderGetter diskHolderGetterFunc) []ReadWriteStats {
	diskStatsMap := make(map[string]ReadWriteStats)
	partitionStatsMap := make(map[string]ReadWriteStats)
	deviceMapperHolderMap := make(map[string]string)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		diskStats, err := statsForDisk(scanner.Text())
		if err != nil {
			continue
		}

		if diskStats.Type == Disk {
			diskStatsMap[diskStats.Name] = *diskStats

			if dmName, err := holderGetter(diskStats.Name, "/sys/class/block/%s/holders/"); err == nil && dmName != "" {
				deviceMapperHolderMap[dmName] = diskStats.Name
			}
		} else {
			partitionStatsMap[diskStats.Name] = *diskStats
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	for _, partitionStats := range partitionStatsMap {

		var diskName string
		var ok bool

		switch partitionStats.Type {
		case Partition:
			diskName = strings.TrimRight(partitionStats.Name,"0123456789")
		case DeviceMapper:
			if diskName, ok = deviceMapperHolderMap[partitionStats.Name]; !ok {
				continue
			}
		default:
			continue
		}

		var diskStats ReadWriteStats
		if diskStats, ok = diskStatsMap[diskName]; !ok {
			continue
		}
		if diskStats.Type == Disk {
			// replace disk statistics by partition or holder stats
			diskStats.Type = partitionStats.Type
			diskStats.Writes = partitionStats.Writes
			diskStats.Reads = partitionStats.Reads
		} else {
			// otherwise, accumulate stats of all partitions and holder if any
			diskStats.Writes += partitionStats.Writes
			diskStats.Reads += partitionStats.Reads
		}
		diskStatsMap[diskName] = diskStats
	}

	return toSlice(diskStatsMap)
}

func getDiskHolder(diskName, pathFormat string) (string, error) {
	/* This returns only the first holder. In practice when using LUKS, there is only one holder */

	holdersDir := fmt.Sprintf(pathFormat, diskName)
	if _, err := os.Stat(holdersDir); os.IsNotExist(err) {
		return "", err
	}

	files, err := os.ReadDir(holdersDir)
	if err != nil {
		return "", err
	}
	for _, file := range files {
		return file.Name(), nil
	}
	return "", nil
}

func statsForDisk(rawStats string) (*ReadWriteStats, error) {
	reader := strings.NewReader(rawStats)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		cols := strings.Fields(scanner.Text())

		name := cols[deviceNameCol]
		deviceType := Unknown
		reads, _ := strconv.ParseUint(cols[readsCol], 10, 64)
		writes, _ := strconv.ParseUint(cols[writesCol], 10, 64)
		

		if scsiDiskRegex.MatchString(name) {
			deviceType = Disk
		} else if scsiPartitionRegex.MatchString(name) {
			deviceType = Partition
		} else if deviceMapperRegex.MatchString(name) {
			deviceType = DeviceMapper
		} else {
			continue
		}

		stats := &ReadWriteStats{
			Name:   name,
			Type:   deviceType,
			Reads:  reads,
			Writes: writes,
		}
		return stats, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("cannot read disk stats")
}

func toSlice(rws map[string]ReadWriteStats) []ReadWriteStats {
	var snapshot []ReadWriteStats
	for _, r := range rws {
		snapshot = append(snapshot, r)
	}
	return snapshot
}
