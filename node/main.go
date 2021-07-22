package main

import (
	"flag"
	"log"
)

func main() {
	log.Println("starts")
	go func() {
		WaitBreak()
		log.Println("shutting down by break begin")
	}()

	flag.Parse()
	Run()
	WaitExit()
	log.Println("shutting down complete.")
}
