package config

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"
)

// ClientTLSConfig holds the paths to the TLS certificate files for client mTLS.
type ClientTLSConfig struct {
	CertFile string // Path to client's certificate file (PEM format)
	KeyFile  string // Path to client's private key file (PEM format)
	CAFile   string // Path to CA's certificate file (PEM format) to verify the server
}

// GetClientTLSConfig returns a ClientTLSConfig based on the current configuration
// If EnableMTLS is false or if any of the required certificate paths are missing,
// it returns nil, which will result in an insecure connection for the client.
func GetClientTLSConfig() *ClientTLSConfig {
	if !EnableMTLS {
		return nil
	}

	// Only proceed if all required paths are specified
	if ClientCert == "" || ClientKey == "" || CACert == "" {
		log.Printf("Warning: Missing required client certificate paths. ClientCert: %v, ClientKey: %v, CACert: %v", ClientCert != "", ClientKey != "", CACert != "")
		return nil
	}

	return &ClientTLSConfig{
		CertFile: ClientCert,
		KeyFile:  ClientKey,
		CAFile:   CACert,
	}
}

// LoadServerTLSConfig creates and returns a *tls.Config for server-side mTLS.
// It loads the server's certificate and key, and the CA certificate for client authentication.
// Returns nil if server mTLS is not enabled or if any certificate paths are missing/invalid.
func LoadServerTLSConfig() *tls.Config {
	if !ServerEnableMTLS {
		log.Println("Server mTLS is not enabled.")
		return nil
	}

	if ServerCertFile == "" || ServerKeyFile == "" || ServerCACertFile == "" {
		log.Printf("Warning: Missing required server certificate paths for mTLS. ServerCertFile: %v, ServerKeyFile: %v, ServerCACertFile: %v", ServerCertFile != "", ServerKeyFile != "", ServerCACertFile != "")
		return nil
	}

	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair(ServerCertFile, ServerKeyFile)
	if err != nil {
		log.Printf("Error loading server certificate/key: %v", err)
		return nil
	}

	// Create a CA certificate pool and add client CA cert
	caCert, err := os.ReadFile(ServerCACertFile)
	if err != nil {
		log.Printf("Error reading server CA certificate: %v", err)
		return nil
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		log.Println("Error: Failed to append client CA certificate to pool.")
		return nil
	}

	// Setup TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert, // Require and verify client certificate
		ClientCAs:    caCertPool,                     // Set CA for client certificate validation
	}

	return tlsConfig
}
