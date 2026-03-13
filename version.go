package main

import (
	"runtime/debug"
	"time"
)

var (
	Version   = "0.0.2"
	Commit    = ""
	BuildDate = ""
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if Commit == "" && len(s.Value) >= 7 {
				Commit = s.Value[:7]
			}
		case "vcs.time":
			if BuildDate == "" {
				if t, err := time.Parse(time.RFC3339, s.Value); err == nil {
					BuildDate = t.UTC().Format("2006-01-02")
				}
			}
		}
	}
	if Commit == "" {
		Commit = "unknown"
	}
	if BuildDate == "" {
		BuildDate = "unknown"
	}
}
