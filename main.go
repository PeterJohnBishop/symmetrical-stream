package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/peterjohnbishop/symmetrical-stream/wsconn"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found, relying on system environment variables")
	}

	host := os.Getenv("HOST")
	if host == "" {
		log.Fatal("HOST environment variable is not set")
	}
	resp, err := http.Get("https://" + host)
	if err != nil {
		fmt.Println("Error fetching the URL:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading the response body:", err)
		return
	}

	fmt.Println(string(body))

	wss := wsconn.ConnectionManager{
		MessageChan: make(chan wsconn.EventMessage),
		ErrorChan:   make(chan error),
	}

	conn, err := wss.Connect()
	if err != nil {
		log.Fatal("Failed to connect to WebSocket server:", err)
	}

	wss.Conn = conn
	go wss.StartListening()

	// Keep the main function running to listen for messages
	select {}
}
