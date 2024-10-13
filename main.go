package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Suhaibinator/SuhaibMessageQueue/config"
	"github.com/Suhaibinator/SuhaibMessageQueue/server"
)

func init() {
	// Custom usage function to display help information
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nAll flags are optional. Environment variables or default values are used if flags are not provided.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Environment Variables:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s: Path to the SQLite database file (default \"%s\")\n", config.ENV_DB_PATH, config.DBPath)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s: Port to listen on (default \"%s\")\n", config.ENV_PORT, config.Port)
	}

	// Check environment variables
	if dbPath, exists := os.LookupEnv(config.ENV_DB_PATH); exists {
		config.DBPath = dbPath
	}
	if port, exists := os.LookupEnv(config.ENV_PORT); exists {
		config.Port = port
	}

	// Check program arguments
	flag.StringVar(&config.DBPath, "dbpath", config.DBPath, "path to the SQLite database file")
	flag.StringVar(&config.Port, "port", config.Port, "port to listen on")
	flag.Parse()
}

func main() {
	// Start the server
	server := server.NewServer(config.Port, config.DBPath)
	server.Start()
}
