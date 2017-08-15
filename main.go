package main

import (
	"flag"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/nhooyr/log"
	"github.com/pkg/errors"
)

func waitSSH(cmd *exec.Cmd, errc chan<- error) {
	err := cmd.Wait()
	pErr, ok := err.(*exec.ExitError)
	if !ok {
		errc <- errors.Wrap(err, "unexpected error waiting for ssh")
		return
	}
	status := pErr.ProcessState.Sys().(syscall.WaitStatus)
	if status.Signal() != os.Kill {
		errc <- errors.Wrap(err, "ssh not killed by us")
	}
}

func main() {
	host := flag.String("host", "", "host to connect to")
	port := flag.String("port", "", "port to connect to")
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

	sshArgs := []string{*host, "-D", *socks5Addr, "-o", "ControlPath=none", "-N"}
	if *port != "" {
		sshArgs = append(sshArgs, "-p", *port)
	}
	ssh := exec.Command("ssh", sshArgs...)
	ssh.Stdin = os.Stdin
	ssh.Stdout = os.Stdout
	ssh.Stderr = os.Stderr
	err := ssh.Start()
	if err != nil {
		log.Fatalf("failed to start ssh: %v", err)
	}

	sshErrc := make(chan error)
	go waitSSH(ssh, sshErrc)

	// Allow some time for SSH to start and report a possible error.
	time.Sleep(10 * time.Millisecond)

	select {
	case err := <-sshErrc:
		log.Fatalf("ssh unexpectedly quit: %v", err)
	default:
	}

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
	case err = <-sshErrc:
		log.Printf("ssh unexpectedly quit: %v", err)
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
