package diskstats

import (
	"bufio"
	"errors"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
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

func TakeSnapshot() []DiskStats {
	diskStatsFile, err := os.Open("/proc/diskstats")
	if err != nil {
		log.Fatal(err)
	}
	defer diskStatsFile.Close()

	var snapshot []DiskStats
	scanner := bufio.NewScanner(diskStatsFile)
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
		reads, _ := strconv.Atoi(cols[3])
		writes, _ := strconv.Atoi(cols[7])
		name := cols[2]
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
