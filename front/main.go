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
	makeServerLabel("DFS", "0.1.0")

	var gmux = NewRouter()
	RegisterRoutes(gmux)
	Run(gmux)

	log.Printf("ready")
	WaitExit()
	log.Println("shutting down complete.")
}
