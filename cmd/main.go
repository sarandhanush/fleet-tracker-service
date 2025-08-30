package main

import (
	"log"

	"fleet-tracker-service/server"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
