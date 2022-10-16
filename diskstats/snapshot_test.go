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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func mockGetDiskHolder(diskName, format string) (string, error) {
	if diskName == "sdf" {
		return "dm-4", nil
	}
	return "", nil
}

func TestTakeSnapshot(t *testing.T) {
	s := `   7       0 loop0 0 0 0 0 0 0 0 0 0 0 0
   7       1 loop1 0 0 0 0 0 0 0 0 0 0 0
   7       2 loop2 0 0 0 0 0 0 0 0 0 0 0
   7       3 loop3 0 0 0 0 0 0 0 0 0 0 0
   7       4 loop4 0 0 0 0 0 0 0 0 0 0 0
   7       5 loop5 0 0 0 0 0 0 0 0 0 0 0
   7       6 loop6 0 0 0 0 0 0 0 0 0 0 0
   7       7 loop7 0 0 0 0 0 0 0 0 0 0 0
 179       0 mmcblk0 133145 53235 6878634 3020910 1544414 1254150 48441345 240124500 0 13439150 243142800
 179       1 mmcblk0p1 80 40 1760 210 1 0 1 0 0 140 210
 179       2 mmcblk0p2 132931 53195 6874482 3020440 1544413 1254150 48441344 240124500 0 13439000 243278260
   8       0 sda 321553 158156 37537568 5961590 50820 94361 10439592 26691430 0 3357150 32650910
   8       1 sda1 321454 158156 37536344 5725790 50820 94361 10439592 26691430 0 3121370 32415240
   8      32 sdc 52147 2738 6494584 913050 28092 1251 6370936 8938800 0 506360 9852970
   8      33 sdc1 52087 2738 6493672 905390 28092 1251 6370936 8938800 0 498700 9892750
   8      16 sdb 5650742 34516 727476416 92732820 1728864 35618 404215912 705303450 0 22944140 798112260
   8      17 sdb1 5650643 34516 727475192 92673920 1728864 35618 404215912 705303450 0 22893010 798071230
   8      16 sdd 982501 110903 37074938 9468348 60870 112203 15682640 17081448 0 2086868 26550788
   8      17 sdd1 369 0 39960 1288 0 0 0 0 0 792 1288
   8      18 sdd2 443 0 53496 1224 0 0 0 0 0 1204 1224
   8      19 sdd3 52 0 2632 76 0 0 0 0 0 76 76
   8      21 sdd5 76541 1090 22221672 979636 8845 2335 8316352 14986056 0 470364 15965544
   8      22 sdd6 904573 109813 14736818 8485644 51818 109868 7366288 2094080 0 1642436 10580612
   8      22 sdd11 904573 109813 1 8485644 51818 109868 1 2094080 0 1642436 10580612
   8      32 sde 2891 192 57743 13523 1085550 14035 982686648 9819204 0 5102050 10209416 0 0 0 0 12140 376689
   8      34 sdf 207814 251309 3670180 1378314 38505 27680 21787544 926421 0 552176 2325026 0 0 0 0 272 20290
 253       4 dm-4 25371 0 206376 195492 1330 0 10640 657392 0 19480 852884 0 0 0 0 0 0
  65     160 sdaa 157371 937 11375913 1617587 8304860 2117223236 17004224768 98631435 0 49649267 100249022 0 0 0 0 0 0
  65     161 sdaa1 157257 937 11371536 1617417 8304860 2117223236 17004224768 98631435 0 49649104 100248853 0 0 0 0 0 0
  65     176 sdab 54244 803 1223811 596585 368 9 3008 1051 0 342387 597828 0 0 0 0 8 191`

	stats := readSnapshot(strings.NewReader(s), mockGetDiskHolder)
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Name < stats[j].Name
	})

	expected := []ReadWriteStats{
		{Name: "sda", Type: Partition, Reads: 37536344, Writes: 10439592},
		{Name: "sdaa", Type: Partition, Reads: 11371536, Writes: 17004224768},
		{Name: "sdab", Type: Disk, Reads: 1223811, Writes: 3008},
		{Name: "sdb", Type: Partition, Reads: 727475192, Writes: 404215912},
		{Name: "sdc", Type: Partition, Reads: 6493672, Writes: 6370936},
		{Name: "sdd", Type: Partition, Reads: 37054579, Writes: 15682641},
		{Name: "sde", Type: Disk, Reads: 57743, Writes: 982686648},
		{Name: "sdf", Type: DeviceMapper, Reads: 206376, Writes: 10640},
	}

	if len(expected) != len(stats) {
		t.Fatalf("Expected %d disks but found %d", len(expected), len(stats))
	}
	for i := 0; i < len(expected); i++ {
		exp := expected[i]
		act := stats[i]

		if exp != act {
			t.Fatalf("Expected %v but found %v", exp, act)
		}
	}
}

func TestGetDiskHolder(t *testing.T) {
	type wantParams struct {
		name         string
		errorMessage string
	}
	tests := []struct {
		name       string
		diskName   string
		holderPath string
		want       wantParams
	}{
		{
			name:       "disk not found",
			diskName:   "sda",
			holderPath: "",
			want: wantParams{
				name:         "",
				errorMessage: "stat /tmp/sys/class/block/sda/holders/: no such file or directory",
			},
		}, {
			name:       "disk found",
			diskName:   "sda",
			holderPath: "/tmp/sys/class/block/sda/holders/dm-0",
			want: wantParams{
				name:         "dm-0",
				errorMessage: "",
			},
		},
	}
	for _, test := range tests {
		err := os.RemoveAll("/tmp/sys")
		if err != nil {
			panic(err)
		}
		t.Run(test.name, func(t *testing.T) {
			if len(test.holderPath) > 0 {
				if err := os.MkdirAll(filepath.Dir(test.holderPath), 0770); err != nil {
					panic(err)
				}
				_, err := os.Create(test.holderPath)
				if err != nil {
					panic(err)
				}
			}
			got, err := getDiskHolder(test.diskName, "/tmp/sys/class/block/%s/holders/")

			if len(test.want.errorMessage) > 0 &&
				test.want.errorMessage != err.Error() {

				t.Fatalf("Expected %v but found %v", test.want.errorMessage, err.Error())
			}

			if test.want.name != got {
				t.Fatalf("Expected %v but found %v", test.want.name, got)
			}

		})
	}
}

func TestStatsForDisk(t *testing.T) {
	type wantParams struct {
		name         string
		deviceType   DeviceType
		errorMessage string
	}
	tests := []struct {
		name string
		line string
		want wantParams
	}{
		{
			name: "disk type",
			line: "8 0 sda 321553 158156 37537568 5961590 50820 94361 10439592 26691430 0 3357150 32650910",
			want: wantParams{
				name:       "sda",
				deviceType: Disk,
			},
		},
		{
			name: "partition type",
			line: "8 17 sdd1 369 0 39960 1288 0 0 0 0 0 792 1288",
			want: wantParams{
				name:       "sdd1",
				deviceType: Partition,
			},
		},
		{
			name: "device mapper type",
			line: "253 4 dm-4 25371 0 206376 195492 1330 0 10640 657392 0 19480 852884 0 0 0 0 0 0",
			want: wantParams{
				name:       "dm-4",
				deviceType: DeviceMapper,
			},
		},
		{
			name: "unknown type",
			line: "7 1 loop1 0 0 0 0 0 0 0 0 0 0 0",
			want: wantParams{
				errorMessage: "cannot read disk stats",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, gotError := statsForDisk(test.line)

			if test.want.errorMessage != "" && test.want.errorMessage != gotError.Error() {
				t.Fatalf("Expected %v but found %v", test.want.errorMessage, gotError.Error())
			}
			if gotError != nil {
				return
			}

			if test.want.name != got.Name {
				t.Fatalf("Expected %v but found %v", test.want.name, got.Name)
			}

			if test.want.deviceType != got.Type {
				t.Fatalf("Expected %v but found %v", test.want.deviceType, got.Type)
			}
		})
	}
}
