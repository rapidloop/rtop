/*

rtop - the remote system monitoring utility

Copyright (c) 2015-17 RapidLoop

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
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
)

func getpass(prompt string) (pass string, err error) {

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
			fmt.Println()
			os.Exit(2)
		}
	}()
	defer func() {
		signal.Stop(sig)
		close(sig)
	}()

	f := bufio.NewWriter(os.Stdout)
	f.Write([]byte(prompt))
	f.Flush()

	passbytes, err := terminal.ReadPassword(0)
	pass = string(passbytes)

	f.Write([]byte("\n"))
	f.Flush()

	return
}

// ParsePemBlock: ref golang.org/x/crypto/ssh/keys.go#ParseRawPrivateKey.
func ParsePemBlock(block *pem.Block) (interface{}, error) {

	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	case "DSA PRIVATE KEY":
		return ssh.ParseDSAPrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("rtop: unsupported key type %q", block.Type)
	}
}

func expandPath(path string) string {

	if len(path) < 2 || path[:2] != "~/" {
		return path
	}

	return strings.Replace(path, "~", currentUser.HomeDir, 1)
}

func addKeyAuth(auths []ssh.AuthMethod, keypath string) []ssh.AuthMethod {
	if len(keypath) == 0 {
		return auths
	}

	keypath = expandPath(keypath)

	// read the file
	pemBytes, err := ioutil.ReadFile(keypath)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	// get first pem block
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		log.Printf("no key found in %s", keypath)
		return auths
	}

	// handle plain and encrypted keyfiles
	if x509.IsEncryptedPEMBlock(block) {
		prompt := fmt.Sprintf("Enter passphrase for key '%s': ", keypath)
		pass, err := getpass(prompt)
		if err != nil {
			return auths
		}
		block.Bytes, err = x509.DecryptPEMBlock(block, []byte(pass))
		if err != nil {
			log.Print(err)
			return auths
		}
		key, err := ParsePemBlock(block)
		if err != nil {
			log.Print(err)
			return auths
		}
		signer, err := ssh.NewSignerFromKey(key)
		if err != nil {
			log.Print(err)
			return auths
		}
		return append(auths, ssh.PublicKeys(signer))
	} else {
		signer, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			log.Print(err)
			return auths
		}
		return append(auths, ssh.PublicKeys(signer))
	}
}

func getAgentAuth() (auth ssh.AuthMethod, ok bool) {
	if sock := os.Getenv("SSH_AUTH_SOCK"); len(sock) > 0 {
		if agconn, err := net.Dial("unix", sock); err == nil {
			ag := agent.NewClient(agconn)
			auth = ssh.PublicKeysCallback(ag.Signers)
			ok = true
		}
	}
	return
}

func addPasswordAuth(user, addr string, auths []ssh.AuthMethod) []ssh.AuthMethod {
	if terminal.IsTerminal(0) == false {
		return auths
	}
	host := addr
	if i := strings.LastIndex(host, ":"); i != -1 {
		host = host[:i]
	}
	prompt := fmt.Sprintf("%s@%s's password: ", user, host)
	passwordCallback := func() (string, error) {
		return getpass(prompt)
	}
	return append(auths, ssh.PasswordCallback(passwordCallback))
}

func tryAgentConnect(user, addr string) (client *ssh.Client) {
	if auth, ok := getAgentAuth(); ok {
		config := &ssh.ClientConfig{
			User: user,
			Auth: []ssh.AuthMethod{auth},
		}
		client, _ = ssh.Dial("tcp", addr, config)
	}

	return
}

func sshConnect(user, addr, keypath string) (client *ssh.Client) {
	// try connecting via agent first
	client = tryAgentConnect(user, addr)
	if client != nil {
		return
	}

	// if that failed try with the key and password methods
	auths := make([]ssh.AuthMethod, 0, 2)
	auths = addKeyAuth(auths, keypath)
	auths = addPasswordAuth(user, addr, auths)

	config := &ssh.ClientConfig{
		User: user,
		Auth: auths,
		HostKeyCallback: func(string, net.Addr, ssh.PublicKey) error {
			return nil
		},
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
