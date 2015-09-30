/*

rtop-bot - remote system monitoring bot

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
	"bufio"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

type Section struct {
	Hostname     string
	Port         int
	User         string
	IdentityFile string
}

func (s *Section) clear() {
	s.Hostname = ""
	s.Port = 0
	s.User = ""
	s.IdentityFile = ""
}

func (s *Section) getFull(name string, def Section) (host string, port int, user, keyfile string) {
	if len(s.Hostname) > 0 {
		host = s.Hostname
	} else if len(def.Hostname) > 0 {
		host = def.Hostname
	}
	if s.Port > 0 {
		port = s.Port
	} else if def.Port > 0 {
		port = def.Port
	}
	if len(s.User) > 0 {
		user = s.User
	} else if len(def.User) > 0 {
		user = def.User
	}
	if len(s.IdentityFile) > 0 {
		keyfile = s.IdentityFile
	} else if len(def.IdentityFile) > 0 {
		keyfile = def.IdentityFile
	}
	return
}

var HostInfo = make(map[string]Section)

func getSshEntry(name string) (host string, port int, user, keyfile string) {

	def := Section{Hostname: name}
	if defcfg, ok := HostInfo["*"]; ok {
		def = defcfg
	}

	if s, ok := HostInfo[name]; ok {
		return s.getFull(name, def)
	}
	for h, s := range HostInfo {
		if ok, err := path.Match(h, name); ok && err == nil {
			return s.getFull(name, def)
		}
	}
	return def.Hostname, def.Port, def.User, def.IdentityFile
}

func parseSshConfig(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("warning: %v", err)
		return false
	}
	defer f.Close()
	update := func(cb func(s *Section)) {}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) > 1 && strings.ToLower(parts[0]) == "host" {
			hosts := parts[1:]
			for _, h := range hosts {
				if _, ok := HostInfo[h]; !ok {
					HostInfo[h] = Section{}
				}
			}
			update = func(cb func(s *Section)) {
				for _, h := range hosts {
					s, _ := HostInfo[h]
					cb(&s)
					HostInfo[h] = s
				}
			}
		}
		if len(parts) == 2 {
			switch strings.ToLower(parts[0]) {
			case "hostname":
				update(func(s *Section) {
					s.Hostname = parts[1]
				})
			case "port":
				if p, err := strconv.Atoi(parts[1]); err == nil {
					update(func(s *Section) {
						s.Port = p
					})
				}
			case "user":
				update(func(s *Section) {
					s.User = parts[1]
				})
			case "identityfile":
				update(func(s *Section) {
					s.IdentityFile = parts[1]
				})
			}
		}
	}
	return true
}
