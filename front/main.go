package main

import (
	"log"
)

func main() {
	log.Println("starts")

	makeServerLabel("DFS", "0.1.0")

	var gmux = NewRouter()
	RegisterRoutes(gmux)
	Run(gmux)

	log.Printf("ready")
	go func() {
		WaitBreak()
		log.Println("shutting down by break begin")
	}()
	WaitExit()
	log.Println("shutting down complete.")
}
