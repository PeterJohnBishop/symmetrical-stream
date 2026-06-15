package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/peterjohnbishop/symmetrical-stream/rtconn"
	"github.com/peterjohnbishop/symmetrical-stream/wsconn"
)

type ServerResponse struct {
	Status  string `json:"Status"`
	Message string `json:"Message"`
}

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
		log.Fatal("Error fetching the URL:", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading the response body:", err)
	}

	var response ServerResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Fatal("Error unmarshaling JSON:", err)
	}

	if response.Status != "Active" {
		log.Fatalf("Server returned non-OK status: %s - %s", response.Status, response.Message)
	}

	wss := wsconn.ConnectionManager{
		MessageChan: make(chan wsconn.EventMessage),
		StatusChan:  make(chan string, 100),
	}

	rtc := rtconn.WebRTCManager{
		WC:             &wss,
		StatusChan:     make(chan string, 100),
		LocalDataChan:  make(chan string, 5),
		RemoteDataChan: make(chan string, 5),
	}
}
