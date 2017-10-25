package main

import (
	"flag"
	"net"
	"os"
	"os/exec"
	"os/signal"

	"github.com/nhooyr/log"
)

func requireRoot() {
	if os.Geteuid() != 0 {
		sudo := exec.Command("sudo", os.Args...)
		attachCmd(sudo)
		// TODO research whether it is possible to use Start here instead and have the terminal wait.
		err := sudo.Run()
		must(err)
		os.Exit(0)
	}
}

func main() {
	requireRoot()

	host := flag.String("host", "", "host to connect to")
	port := flag.String("port", "", "port to connect to")
	socks5Addr := flag.String("D", "localhost:5030", "socks5 listening address as [addr:]port")
	network := flag.String("net", "Wi-Fi", "network to configure to use SOCKS5 proxy")
	flag.Parse()

	// TODO add this into the log package.
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
	attachCmd(ssh)
	err := ssh.Start()
	must(err)

	errc := make(chan error, 1)
	go func() {
		errc <- ssh.Wait()
	}()

	socks5Host, socks5Port, err := net.SplitHostPort(*socks5Addr)
	must(err)
	if socks5Host == "" {
		socks5Host = "localhost"
	}

	networkSetup := exec.Command("networksetup", "-setsocksfirewallproxy", *network, socks5Host, socks5Port)
	attachCmd(networkSetup)
	err = networkSetup.Run()
	must(err)

	log.Print("initialized")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	select {
	case <-c:
		err := ssh.Process.Kill()
		if err != nil {
			log.Printf("error killing ssh: %v", err)
		}
	case err = <-errc:
		log.Printf("ssh unexpectedly quit: %v", err)
	}

	networkSetup = exec.Command("networksetup", "-setsocksfirewallproxystate", *network, "off")
	attachCmd(networkSetup)
	err = networkSetup.Run()
	must(err)
}

func attachCmd(cmd *exec.Cmd) {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
