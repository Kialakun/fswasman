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
	start_script    = flag.String("start", "default", "path to file to execute on start")
	stop_script     = flag.String("stop", "default", "path to file to execute on stop")
	onchange_script = flag.String("onchange", "default", "path to file to execute on file system change")
	version         = "v0.1.2"
	splash_screen   = `
	
|	File System Wasman ` + version + `
|	----------------------------------				
|	Wasman blo file system.
|			
|	By: Kialakun Galgal  https://github.com/Kialakun
`
)

func batcher(q chan string) {
	prev_task := ""
	counter := 0
	for {
		select {
		case task := <-q:
			if task != prev_task {
				log.Println("change detected ...")
				log.Println("modified file:", task)
				log.Println("executing script...")
				executeScript(*onchange_script)
			} else {
				counter++
			}
			prev_task = task
			if counter > 2 {
				// reset task and counter
				counter = 0
				prev_task = ""
			}
		}
	}
}

// function to handle execting bash scripts
func executeScript(script string) {
	var cmd *exec.Cmd

	// if not set, run a defualt script
	if script == "default" {
		cmd = exec.Command("echo", "Done.")
	} else {
		cmd = exec.Command("/bin/bash", script)
	}
	stdout, err := cmd.Output()
	if err != nil {
		log.Println(string(stdout))
		log.Fatal("An error occured while executing script ", script, " err:", err)
	}
	log.Println(string(stdout))
}

// stop watching on "ctrl+c"
func stop(c chan os.Signal, done chan bool, script string) {
	<-c
	log.Println("Executing stop script...", script)
	executeScript(script)

	// stop main
	done <- true
}

// function for starting a watcher instance
func start(watcher *fsnotify.Watcher, start string, onchange string, q chan string) {
	log.Println("Executing start script...", start)
	executeScript(start)
	for {
		log.Println("waiting for changes...")
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				// add task to que
				q <- event.String() + ":" + event.Name
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

	// create task que
	q := make(chan string, 1)

	// start go routine for listening and handling changes
	go start(watcher, *start_script, *onchange_script, q)

	// start stop wathcer listener on seperate go routine
	go stop(c, done, *stop_script)

	// start batcher to automatically clear previous events after exec_delay
	go batcher(q)

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
