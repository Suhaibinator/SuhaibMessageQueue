package config

// TLSConfig holds the paths to the TLS certificate files for mTLS.
type TLSConfig struct {
	CertFile string // Path to client's certificate file (PEM format)
	KeyFile  string // Path to client's private key file (PEM format)
	CAFile   string // Path to CA's certificate file (PEM format) to verify the server
}

// GetTLSConfig returns a TLSConfig based on the current configuration
// If EnableMTLS is false or if any of the required certificate paths are missing,
// it returns nil, which will result in an insecure connection.
func GetTLSConfig() *TLSConfig {
	if !EnableMTLS {
		return nil
	}

	// Only proceed if all required paths are specified
	if ClientCert == "" || ClientKey == "" || CACert == "" {
		return nil
	}

	return &TLSConfig{
		CertFile: ClientCert,
		KeyFile:  ClientKey,
		CAFile:   CACert,
	}
}
