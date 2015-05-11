package main

import (
	"fmt"
	"time"
)

func fmtUptime(stats *Stats) string {
	dur := stats.Uptime
	dur = dur - (dur % time.Second)
	var days int
	for dur.Hours() > 24.0 {
		days++
		dur -= 24 * time.Hour
	}
	s1 := dur.String()
	s2 := ""
	if days > 0 {
		s2 = fmt.Sprintf("%dd ", days)
	}
	for _, ch := range s1 {
		s2 += string(ch)
		if ch == 'h' || ch == 'm' {
			s2 += " "
		}
	}
	return s2
}

func fmtBytes(val uint64) string {
	if val < 1024 {
		return fmt.Sprintf("%d bytes", val)
	} else if val < 1024*1024 {
		return fmt.Sprintf("%6.2f KiB", float64(val)/1024.0)
	} else if val < 1024*1024*1024 {
		return fmt.Sprintf("%6.2f MiB", float64(val)/1024.0/1024.0)
	} else {
		return fmt.Sprintf("%6.2f GiB", float64(val)/1024.0/1024.0/1024.0)
	}
}
