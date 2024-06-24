package config

import (
	"flag"
	"fmt"
	"os"
)

const (
	ENV_DB_PATH = "DB_PATH"
	ENV_PORT    = "PORT"
)

var (
	// DBPath is the path to the SQLite database file
	DBPath = "/db/dbtest.db"
	Port   = "80097"
)

func init() {
	// Custom usage function to display help information
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nAll flags are optional. Environment variables or default values are used if flags are not provided.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Environment Variables:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s: Path to the SQLite database file (default \"%s\")\n", ENV_DB_PATH, DBPath)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s: Port to listen on (default \"%s\")\n", ENV_PORT, Port)
	}

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
