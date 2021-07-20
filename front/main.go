package main

import (
	"log"
)

func main() {
	log.Println("starts")
	go func() {
		WaitBreak()
		log.Println("shutting down by break begin")
	}()

	makeServerLabel("DFS", "0.1.0")

	var gmux = NewRouter()
	RegisterRoutes(gmux)
	Run(gmux)
	WaitExit()
	log.Println("shutting down complete.")
}