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
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
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

type DiskStats struct {
	Name        string
	IdleTime    int
	CommandType string
	Reads       int
	Writes      int
	SpinDownAt  time.Time
	SpinUpAt    time.Time
	LastIoAt    time.Time
	SpunDown    bool
}

var scsiDiskRegex *regexp.Regexp

func init() {
	scsiDiskRegex = regexp.MustCompile("sd[a-z]$")
}

func Snapshot() []DiskStats {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	return ReadSnapshot(f)
}

func ReadSnapshot(r io.Reader) []DiskStats {
	var snapshot []DiskStats
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		diskStats, err := statsForDisk(scanner.Text())
		if err == nil {
			snapshot = append(snapshot, *diskStats)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return snapshot
}

func statsForDisk(rawStats string) (*DiskStats, error) {
	reader := strings.NewReader(rawStats)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		cols := strings.Fields(scanner.Text())
		name := cols[deviceNameCol]
		reads, _ := strconv.Atoi(cols[readsCol])
		writes, _ := strconv.Atoi(cols[writesCol])
		if !scsiDiskRegex.MatchString(name) {
			return nil, errors.New("disk is a partition")
		}
		stats := &DiskStats{
			Name:   name,
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
