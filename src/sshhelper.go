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
	"bufio"
	"bytes"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func addKeyAuth(auths []ssh.AuthMethod, keypath string) []ssh.AuthMethod {
	if len(keypath) == 0 {
		return auths
	}
	pemBytes, err := ioutil.ReadFile(keypath)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	return append(auths, ssh.PublicKeys(signer))
}

func addAgentAuth(auths []ssh.AuthMethod) []ssh.AuthMethod {
	if sock := os.Getenv("SSH_AUTH_SOCK"); len(sock) > 0 {
		if agconn, err := net.Dial("unix", sock); err == nil {
			ag := agent.NewClient(agconn)
			auths = append(auths, ssh.PublicKeysCallback(ag.Signers))
		}
	}
	return auths
}

func passwordCallback() (pass string, err error) {

	tstate, err := terminal.GetState(0)
	if err != nil {
		return
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		quit := false
		for _ = range sig {
			quit = true
			break
		}
		terminal.Restore(0, tstate)
		if quit {
			os.Exit(2)
		}
	}()
	defer func() {
		signal.Stop(sig)
		close(sig)
	}()

	f := bufio.NewWriter(os.Stdout)
	f.Write([]byte("Password: "))
	f.Flush()

	passbytes, err := terminal.ReadPassword(0)
	pass = string(passbytes)

	f.Write([]byte("\n"))
	f.Flush()

	return
}

func addPasswordAuth(auths []ssh.AuthMethod) []ssh.AuthMethod {
	if terminal.IsTerminal(0) == false {
		return auths
	}
	return append(auths, ssh.PasswordCallback(passwordCallback))
}

func sshConnect(user, addr, keypath string) (client *ssh.Client) {
	auths := make([]ssh.AuthMethod, 0, 2)
	auths = addAgentAuth(auths)
	auths = addKeyAuth(auths, keypath)
	auths = addPasswordAuth(auths)

	config := &ssh.ClientConfig{
		User: user,
		Auth: auths,
	}
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	return
}

func runCommand(client *ssh.Client, command string) (stdout string, err error) {
	session, err := client.NewSession()
	if err != nil {
		//log.Print(err)
		return
	}
	defer session.Close()

	var buf bytes.Buffer
	session.Stdout = &buf
	err = session.Run(command)
	if err != nil {
		//log.Print(err)
		return
	}
	stdout = string(buf.Bytes())

	return
}
