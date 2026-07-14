package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/peterjohnbishop/symmetrical-stream/signaling"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	signalingServer, err := signaling.ConnectToSignalingServer()
	if err != nil {
		log.Printf("error connecting to the signaling server: %s", err.Error())
	}
	_ = signalingServer // placeholder
}
