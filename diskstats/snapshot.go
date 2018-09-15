package diskstats

import (
	"bufio"
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

func TakeSnapshot() []DiskStats {
	diskStatsFile, err := os.Open("/proc/diskstats")
	if err != nil {
		log.Fatal(err)
	}
	defer diskStatsFile.Close()

	var snapshot []DiskStats
	scanner := bufio.NewScanner(diskStatsFile)
	for scanner.Scan() {
		diskStats := statsForDisk(scanner.Text())
		if diskStats != nil {
			snapshot = append(snapshot, *diskStats)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return snapshot
}

func statsForDisk(rawStats string) *DiskStats {
	reader := strings.NewReader(rawStats)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		cols := strings.Fields(scanner.Text())
		reads, _ := strconv.Atoi(cols[3])
		writes, _ := strconv.Atoi(cols[7])
		name := cols[2]
		r := regexp.MustCompile("sd[a-z]$")
		if !r.MatchString(name) {
			return nil
		}
		stats := &DiskStats{
			Name:   name,
			Reads:  reads,
			Writes: writes,
		}
		return stats
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return nil
}
