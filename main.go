package main

import (
	"flag"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/nhooyr/log"
	"github.com/pkg/errors"
)

func waitSSH(cmd *exec.Cmd, done chan struct{}) error {
	defer close(done)
	err := cmd.Wait()
	pErr, ok := err.(*exec.ExitError)
	if !ok {
		return errors.Wrap(err, "unexpected error waiting for ssh")
	}
	status := pErr.ProcessState.Sys().(syscall.WaitStatus)
	if status.Signal() != os.Kill {
		return errors.Wrap(err, "ssh not killed by us")
	}
	return nil
}

func main() {
	host := flag.String("host", "", "host to connect to")
	socks5Addr := flag.String("D", "localhost:5030", "socks5 listening address (addr:port)")
	network := flag.String("net", "Wi-Fi", "network to configure to use SOCKS5 proxy")
	flag.Parse()

	log := log.Logger{
		Out: log.LineWriter{
			Out: &log.AtomicWriter{Out: os.Stderr},
		},
	}

	if *host == "" {
		log.Print("please provide a host")
		flag.Usage()
		os.Exit(1)
	}

	ssh := exec.Command("ssh", *host, "-D", *socks5Addr, "-o", "ControlPath=none", "-N")
	err := ssh.Start()
	if err != nil {
		log.Fatalf("failed to start ssh: %v", err)
	}

	sshDone := make(chan struct{})
	go func() {
		err := waitSSH(ssh, sshDone)
		if err != nil {
			log.Printf("error waiting for ssh: %v", err)
		}
	}()

	socks5Host, socks5Port, err := net.SplitHostPort(*socks5Addr)
	if err != nil {
		log.Fatalf("failed to split host port socks5Addr: %v", err)
	}
	if socks5Host == "" || socks5Host == "*" {
		socks5Host = "localhost"
	}

	networksetup := exec.Command("sudo", "networksetup", "-setsocksfirewallproxy", *network, socks5Host, socks5Port)
	networksetup.Stdin = os.Stdin
	networksetup.Stdout = os.Stdout
	networksetup.Stderr = os.Stderr
	err = networksetup.Run()
	if err != nil {
		log.Fatalf("failed to set socks5 proxy: %v", err)
	}

	log.Print("initialized")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	select {
	case <-c:
		err := ssh.Process.Kill()
		if err != nil {
			log.Printf("erorr killing ssh: %v", err)
		}
	case <-sshDone:
		log.Printf("ssh unexpectedly quit, state: %v", ssh.ProcessState)
	}

	networksetup = exec.Command("sudo", "networksetup", "-setsocksfirewallproxystate", *network, "off")
	networksetup.Stdin = os.Stdin
	networksetup.Stdout = os.Stdout
	networksetup.Stderr = os.Stderr
	err = networksetup.Run()
	if err != nil {
		log.Fatalf("failed to turn off socks5 proxy: %v", err)
	}
}
