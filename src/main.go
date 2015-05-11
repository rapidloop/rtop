/*

rtop - the remote system monitoring utility

Copyright (c) 2015 RapidLoop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const VERSION = "1.0"
const DEFAULT_REFRESH = 5 // default refresh interval in seconds

//----------------------------------------------------------------------------
// Command-line processing

func usage(code int) {
	fmt.Printf(
`rtop %s - (c) 2015 RapidLoop - MIT Licensed - http://rtop-monitor.org
rtop monitors server statistics over an ssh connection

Usage: rtop [-i private-key-file] [user@]host[:port] [interval]

	-i private-key-file
		PEM-encoded private key file to use (default: ~/.ssh/id_rsa if present)
	[user@]host[:port]
		the SSH server to connect to, with optional username and port
	interval
		refresh interval in seconds (default: %d)

`, VERSION, DEFAULT_REFRESH)
	os.Exit(code)
}

func shift(q []string) (ok bool, val string, qnew []string) {
	if len(q) > 0 {
		ok = true
		val = q[0]
		qnew = q[1:]
	}
	return
}

func parseCmdLine() (key, username, addr string, interval time.Duration) {
	ok, arg, args := shift(os.Args)
	var argKey, argHost, argInt string
	for ok {
		ok, arg, args = shift(args)
		if !ok {
			break
		}
		if arg == "-h" || arg == "--help" || arg == "--version" {
			usage(0)
		}
		if arg == "-i" {
			ok, argKey, args = shift(args)
			if !ok {
				usage(1)
			}
		} else if len(argHost) == 0 {
			argHost = arg
		} else if len(argInt) == 0 {
			argInt = arg
		} else {
			usage(1)
		}
	}
	if len(argHost) == 0 || argHost[0] == '-' {
		usage(1)
	}

	// key
	usr, err := user.Current()
	if err != nil {
		log.Print(err)
		usage(1)
	}
	if len(argKey) == 0 {
		key = filepath.Join(usr.HomeDir, ".ssh", "id_rsa")
		if _, err := os.Stat(key); os.IsNotExist(err) {
			key = ""
		}
	} else {
		key = argKey
	}
	// username, addr
	if i := strings.Index(argHost, "@"); i != -1 {
		username = argHost[:i]
		if i+1 >= len(argHost) {
			usage(1)
		}
		addr = argHost[i+1:]
	} else {
		username = usr.Username
		addr = argHost
	}
	if i := strings.Index(addr, ":"); i == -1 {
		addr += ":22"
	}
	// interval
	if len(argInt) == 0 {
		interval = DEFAULT_REFRESH * time.Second
	} else {
		i, err := strconv.ParseUint(argInt, 10, 64)
		if err != nil {
			log.Print(err)
			usage(1)
		}
		interval = time.Duration(i) * time.Second
	}

	return
}

//----------------------------------------------------------------------------

func main() {

	log.SetPrefix("rtop: ")
	log.SetFlags(0)

	keyPath, username, addr, interval := parseCmdLine()

	client := sshConnect(username, addr, keyPath)

	// the loop
	showStats(client)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	timer := time.Tick(interval)
	done := false
	for !done {
		select {
		case <-sig:
			done = true
		case <-timer:
			showStats(client)
		}
	}
}

func showStats(client *ssh.Client) {
	stats := Stats{}
	getAllStats(client, &stats)
	used := stats.MemTotal - stats.MemFree - stats.MemBuffers - stats.MemCached
	fmt.Printf(
`%s%s%s%s up %s%s%s

Load:
    %s%s %s %s%s

Processes:
    %s%s%s running of %s%s%s total

Memory:
    free    = %s%s%s
    used    = %s%s%s
    buffers = %s%s%s
    cached  = %s%s%s
    swap    = %s%s%s free of %s%s%s

`,
		escClear,
		escBrightWhite, stats.Hostname, escReset,
		escBrightWhite, fmtUptime(&stats), escReset,
		escBrightWhite, stats.Load1, stats.Load5, stats.Load10, escReset,
		escBrightWhite, stats.RunningProcs, escReset,
		escBrightWhite, stats.TotalProcs, escReset,
		escBrightWhite, fmtBytes(stats.MemFree), escReset,
		escBrightWhite, fmtBytes(used), escReset,
		escBrightWhite, fmtBytes(stats.MemBuffers), escReset,
		escBrightWhite, fmtBytes(stats.MemCached), escReset,
		escBrightWhite, fmtBytes(stats.SwapFree), escReset,
		escBrightWhite, fmtBytes(stats.SwapTotal), escReset,
	)
	if len(stats.FSInfos) > 0 {
		fmt.Println("Filesystems:")
		for _, fs := range stats.FSInfos {
			fmt.Printf("    %s%8s%s: %s%s%s free of %s%s%s\n",
				escBrightWhite, fs.MountPoint, escReset,
				escBrightWhite, fmtBytes(fs.Free), escReset,
				escBrightWhite, fmtBytes(fs.Used+fs.Free), escReset,
			)
		}
		fmt.Println()
	}
	if len(stats.NetIntf) > 0 {
		fmt.Println("Network Interfaces:")
		keys := make([]string, 0, len(stats.NetIntf))
		for intf := range stats.NetIntf {
			keys = append(keys, intf)
		}
		sort.Strings(keys)
		for _, intf := range keys {
			info := stats.NetIntf[intf]
			fmt.Printf("    %s%s%s - %s%s%s, %s%s%s\n",
				escBrightWhite, intf, escReset,
				escBrightWhite, info.IPv4, escReset,
				escBrightWhite, info.IPv6, escReset,
			)
			fmt.Printf("      rx = %s%s%s, tx = %s%s%s\n",
				escBrightWhite, fmtBytes(info.Rx), escReset,
				escBrightWhite, fmtBytes(info.Tx), escReset,
			)
			fmt.Println()
		}
		fmt.Println()
	}
}

const (
	escClear       = "\033[H\033[2J"
	escRed         = "\033[31m"
	escReset       = "\033[0m"
	escBrightWhite = "\033[37;1m"
)
