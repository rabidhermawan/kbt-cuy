package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"
)

var (
	MidtransCore *coreapi.Client
	MidtransSnap *snap.Client
)

func LoadConfig() {
	// Load .env file from the current directory
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Can't find .env file, using environment variables from system")
	}

	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	clientKey := os.Getenv("MIDTRANS_CLIENT_KEY")

	var s snap.Client
	s.New(serverKey, midtrans.Sandbox)
	MidtransSnap = &s

	var c coreapi.Client
	c.New(serverKey, midtrans.Sandbox)
	MidtransCore = &c

	// This is not a secure way to expose client key to frontend.
	// We are setting it to os env for the handler to pick it up.
	os.Setenv("MIDTRANS_CLIENT_KEY_FRONTEND", clientKey)
}
