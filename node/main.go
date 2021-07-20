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

	Run()
	WaitExit()
	log.Println("shutting down complete.")
}
