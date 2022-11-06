package config

import (
	"log"
	"os"
)

// SQLUser secret
const SQLUser = ""

// SQLPassword secret
const SQLPassword = ""

// SQLDb secret
const SQLDb = "microservices"

// SQLTableDebug secret
const SQLTableDebug = ""

// SQLTable secret
const SQLTable = ""

type PrimusConfig struct {
	PrimusHost     string
	PrimusPort     string
	PrimusUser     string
	PrimusPassword string
}

func GetPrimusConfig() PrimusConfig {

	host, exists := os.LookupEnv("HOST")
	if !exists {
		log.Fatal("variable HOST not exists, nothing to do")
	}
	port, exists := os.LookupEnv("PORT")
	if !exists {
		log.Fatal("variable PORT not exists, nothing to do")
	}
	return PrimusConfig{
		PrimusHost:     host,
		PrimusPort:     port,
		PrimusUser:     "",
		PrimusPassword: "",
	}
}
