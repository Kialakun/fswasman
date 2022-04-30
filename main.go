package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/fsnotify/fsnotify"
)

var (
	start_script    = flag.String("start", "start.sh", "path to file to execute on start -> default filename 'start.sh'")
	stop_script     = flag.String("stop", "stop.sh", "path to file to execute on stop -> default filename 'stop.sh'")
	onchange_script = flag.String("onchange", "onchange", "path to file to execute on file system change 'onchange.sh'")
	version         = "v0.0.6"
	splash_screen   = `
			File System Wasman ` + version + `
			----------------------------------				
			Wasman blo file system.
			
			By: GRIM
	`
)

// stop watching on "ctrl+c"
func stop(c chan os.Signal, done chan bool, script string) {
	<-c
	log.Println("Executing stop script...", script)
	cmd := exec.Command("/bin/bash", script)
	stdout, err := cmd.Output()
	if err != nil {
		log.Fatal("An error occured while executing stop script, err:", err)
	}
	log.Println(string(stdout))
	log.Println("Stopping watcher.")
	done <- true
}

// function for starting a watcher instance
func start(watcher *fsnotify.Watcher, start string, onchange string, done chan bool) {
	log.Println("Executing start script...", start)
	cmd := exec.Command("/bin/bash", start)
	stdout, err := cmd.Output()
	if err != nil {
		log.Fatal("An error occured while executing stop script, err:", err)
	}
	log.Println(stdout)
	for {
		select {
		case <-done:
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Println("change detected...")
			log.Println("event:", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("modified file:", event.Name)
				log.Println("executing script...")
				cmd := exec.Command("/bin/bash", onchange)
				stdout, err := cmd.Output()
				if err != nil {
					log.Fatal("An error occured while execting bash script. err:", err)
				}
				log.Println(string(stdout))
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}

func main() {

	// get command line args
	flag.Parse()
	DIRS := flag.Args()

	// create a watcher instance
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Watcher could not initate. err:", err)
		return
	}
	defer watcher.Close()
	log.Println(splash_screen + "\n")
	log.Println("Watcher started.")

	// create channel to close program
	done := make(chan bool, 1)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// start go routine for listening and handling changes
	log.Println("waiting for changes...")
	go start(watcher, *start_script, *onchange_script, done)

	// start stop wathcer listener on seperate go routine
	go stop(c, done, *stop_script)

	// star watching directories
	for _, dir := range DIRS {
		err = watcher.Add(dir)
		if err != nil {
			log.Fatal("Error while watching", dir, ". err:", err)
		}
	}
	// shutdown
	<-done
	log.Println("Stopped.")
}