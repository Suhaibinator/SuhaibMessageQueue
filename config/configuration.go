package config

import (
	"flag"
	"os"
)

const (
	ENV_DB_PATH = "DB_PATH"
	ENV_PORT    = "PORT"
)

var (
	// DBPath is the path to the SQLite database file
	DBPath = "dbtest.db"
	Port   = "8080"
)

func init() {
	// Check environment variables
	if dbPath, exists := os.LookupEnv(ENV_DB_PATH); exists {
		DBPath = dbPath
	}
	if port, exists := os.LookupEnv(ENV_PORT); exists {
		Port = port
	}

	// Check program arguments
	flag.StringVar(&DBPath, "dbpath", DBPath, "path to the SQLite database file")
	flag.StringVar(&Port, "port", Port, "port to listen on")
	flag.Parse()
}