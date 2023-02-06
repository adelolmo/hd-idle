package main

import (
	"testing"
	"time"
)

func TestIntervalWithZeroSecondsIdle(t *testing.T) {
	confs := []DeviceConf{{
		Name:        "test",
		GivenName:   "test",
		Idle:        0,
		CommandType: "ata",
	}}
	interval := poolInterval(confs)
	if interval != defaultIdleTime/10 {
		t.Fatalf("interval should be the default. it was %d", interval)
	}
}

func TestIntervalWith300SecondsIdle(t *testing.T) {
	confs := []DeviceConf{{
		Name:        "test",
		GivenName:   "test",
		Idle:        300 * time.Second,
		CommandType: "ata",
	}}
	interval := poolInterval(confs)
	if interval != 30*time.Second {
		t.Fatalf("interval should be the 30s. it was %v", interval)
	}
}
