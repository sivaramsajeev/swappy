package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

const (
	checkInterval = 5 * time.Second
)

type command struct {
	runningPath string
	newPath     string
	args        []string
}

func Start() {
	cmd := parseArgs()
	cmd.run()
}

func parseArgs() *command {
	if len(os.Args) < 2 {
		log.Fatal("USAGE: ./swappy myCmd myArgs")
	}
	return &command{
		runningPath: os.Args[1],
		newPath:     getNewpath(),
		args:        os.Args[2:],
	}
}

func getNewpath() string {
	newPath := os.Getenv("NEW_BIN_PATH")
	if newPath == "" {
		return fmt.Sprintf("%s-nightly", os.Args[1])
	}
	return newPath
}

func (c *command) run() {
	change := make(chan struct{})
	c.watch(change)
	c.exec(change)
}

func (c *command) watch(change chan struct{}) {
	go func(ch chan struct{}) {
		for {
			if c.isBinChanged() {
				change <- struct{}{}
			}
			time.Sleep(checkInterval)
		}
	}(change)
}

func (c *command) isBinChanged() bool {
	fileInfo, err := os.Stat(c.newPath)
	if err != nil {
		return false
	}
	lastCheck := time.Now().Add(checkInterval * -1)
	return fileInfo.ModTime().After(lastCheck)
}

func (c *command) replaceBin() {
	if err := os.Rename(c.newPath, c.runningPath); err != nil {
		log.Fatal("Error moving file", err)
	}
}

func (c *command) exec(change chan struct{}) {
	for {
		cmd := c.getCommand()
		go func(change chan struct{}) {
			<-change
			log.Println("File change detected")
			c.replaceBin()
			for err := cmd.Process.Kill(); err != nil; {
			}
		}(change)

		cmd.Run()
	}
}

func (c *command) getCommand() *exec.Cmd {
	cmd := exec.Command(c.runningPath, c.args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}
