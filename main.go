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
		fmt.Fprintf(flag.CommandLine.Output(), "  %s: Path to client certificate file for mTLS\n", config.ENV_CLIENT_CERT)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s: Path to client key file for mTLS\n", config.ENV_CLIENT_KEY)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s: Path to CA certificate file for mTLS (client to verify server)\n", config.ENV_CA_CERT)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s: Enable client mTLS authentication ('true' or 'false')\n", config.ENV_ENABLE_MTLS)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s: Enable server mTLS authentication ('true' or 'false')\n", config.ENV_SERVER_ENABLE_MTLS)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s: Path to server certificate file for mTLS\n", config.ENV_SERVER_CERT_FILE)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s: Path to server key file for mTLS\n", config.ENV_SERVER_KEY_FILE)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s: Path to CA certificate file for mTLS (server to verify client)\n", config.ENV_SERVER_CA_CERT_FILE)
	}

	// Check environment variables
	if dbPath, exists := os.LookupEnv(config.ENV_DB_PATH); exists {
		config.DBPath = dbPath
	}
	if port, exists := os.LookupEnv(config.ENV_PORT); exists {
		config.Port = port
	}
	if clientCert, exists := os.LookupEnv(config.ENV_CLIENT_CERT); exists {
		config.ClientCert = clientCert
	}
	if clientKey, exists := os.LookupEnv(config.ENV_CLIENT_KEY); exists {
		config.ClientKey = clientKey
	}
	if caCert, exists := os.LookupEnv(config.ENV_CA_CERT); exists {
		config.CACert = caCert
	}
	if enableMTLS, exists := os.LookupEnv(config.ENV_ENABLE_MTLS); exists {
		config.EnableMTLS = enableMTLS == "true" // Client-side
	}
	if serverEnableMTLS, exists := os.LookupEnv(config.ENV_SERVER_ENABLE_MTLS); exists {
		config.ServerEnableMTLS = serverEnableMTLS == "true"
	}
	if serverCertFile, exists := os.LookupEnv(config.ENV_SERVER_CERT_FILE); exists {
		config.ServerCertFile = serverCertFile
	}
	if serverKeyFile, exists := os.LookupEnv(config.ENV_SERVER_KEY_FILE); exists {
		config.ServerKeyFile = serverKeyFile
	}
	if serverCACertFile, exists := os.LookupEnv(config.ENV_SERVER_CA_CERT_FILE); exists {
		config.ServerCACertFile = serverCACertFile
	}

	// Check program arguments
	// Note: Command-line flag values will override environment variable settings for the corresponding configuration options.
	flag.StringVar(&config.DBPath, "dbpath", config.DBPath, "path to the SQLite database file")
	flag.StringVar(&config.Port, "port", config.Port, "port to listen on")
	flag.StringVar(&config.ClientCert, "client-cert", config.ClientCert, "path to client certificate file for mTLS")
	flag.StringVar(&config.ClientKey, "client-key", config.ClientKey, "path to client key file for mTLS")
	flag.StringVar(&config.CACert, "ca-cert", config.CACert, "path to CA certificate file for mTLS (client to verify server)")
	clientEnableMTLSFlag := flag.Bool("enable-mtls", config.EnableMTLS, "enable client-side mTLS authentication") // Renamed for clarity

	// Server mTLS flags
	flag.StringVar(&config.ServerCertFile, "server-cert", config.ServerCertFile, "path to server certificate file for mTLS")
	flag.StringVar(&config.ServerKeyFile, "server-key", config.ServerKeyFile, "path to server key file for mTLS")
	flag.StringVar(&config.ServerCACertFile, "server-ca-cert", config.ServerCACertFile, "path to CA certificate file for mTLS (server to verify client)")
	serverEnableMTLSFlag := flag.Bool("server-enable-mtls", config.ServerEnableMTLS, "enable server-side mTLS authentication")

	flag.Parse()

	// Update EnableMTLS values from flags
	config.EnableMTLS = *clientEnableMTLSFlag       // For client
	config.ServerEnableMTLS = *serverEnableMTLSFlag // For server
}

func main() {
	// Start the server
	server := server.NewServer(config.Port, config.DBPath)
	server.Start()
}
