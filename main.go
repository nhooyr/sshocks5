package main

import (
	"os"
	"os/exec"
	"os/signal"

	"github.com/nhooyr/log"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %v <host>", os.Args[0])
	}
	ssh := exec.Command("ssh", os.Args[1], "-D localhost:5030", "-o ControlPath=none", "-N")
	err := ssh.Start()
	if err != nil {
		log.Fatalf("failed to start ssh: %v", err)
	}

	sshDone := make(chan struct{})
	go func() {
		defer close(sshDone)
		err := ssh.Wait()
		if err != nil {
			log.Printf("failed to wait for ssh: %v", err)
		}
	}()

	networksetup := exec.Command("networksetup", "-setsocksfirewallproxystate", "Wi-Fi", "off")
	err = networksetup.Run()
	if err != nil {
		log.Fatalf("failed to run networksetup: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	select {
	case <-c:
		err := ssh.Process.Kill()
		if err != nil {
			log.Printf("erorr killing ssh: %v", err)
		}
	case <-sshDone:
		log.Print("ssh unexpectedly quit, state: %v", ssh.ProcessState)
	}
}
