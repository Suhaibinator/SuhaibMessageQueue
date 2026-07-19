// Package config loads and validates configuration for the SMQ server process.
package config

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
)

const (
	EnvDBPath            = "DB_PATH"
	EnvPort              = "PORT"
	EnvServerEnableMTLS  = "SERVER_ENABLE_MTLS"
	EnvServerCertFile    = "SERVER_CERT_FILE"
	EnvServerKeyFile     = "SERVER_KEY_FILE"
	EnvServerCACertFile  = "SERVER_CA_CERT_FILE"
	EnvAllowRemoteWrites = "ALLOW_REMOTE_WRITES"

	defaultDBPath         = "./dbtest.db"
	defaultPort           = "8097"
	defaultMaxMessageSize = 1024 * 1024 * 1024
)

// LookupEnv matches os.LookupEnv and makes configuration loading testable.
type LookupEnv func(string) (string, bool)

// Config is the complete validated server configuration.
type Config struct {
	DBPath            string
	Port              string
	AllowRemoteWrites bool
	MaxRecvMsgSize    int
	MaxSendMsgSize    int
	TLS               TLSConfig
}

// TLSConfig contains server-side mutual TLS settings.
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
	CAFile   string
}

// Load reads environment variables, applies command-line overrides, and
// validates the resulting configuration.
func Load(args []string, lookup LookupEnv) (Config, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}

	cfg := Config{
		DBPath:            stringEnv(lookup, EnvDBPath, defaultDBPath),
		Port:              stringEnv(lookup, EnvPort, defaultPort),
		AllowRemoteWrites: true,
		MaxRecvMsgSize:    defaultMaxMessageSize,
		MaxSendMsgSize:    defaultMaxMessageSize,
		TLS: TLSConfig{
			CertFile: stringEnv(lookup, EnvServerCertFile, ""),
			KeyFile:  stringEnv(lookup, EnvServerKeyFile, ""),
			CAFile:   stringEnv(lookup, EnvServerCACertFile, ""),
		},
	}

	var err error
	if cfg.AllowRemoteWrites, err = boolEnv(lookup, EnvAllowRemoteWrites, true); err != nil {
		return Config{}, err
	}
	if cfg.TLS.Enabled, err = boolEnv(lookup, EnvServerEnableMTLS, false); err != nil {
		return Config{}, err
	}

	flags := flag.NewFlagSet("smq", flag.ContinueOnError)
	flags.StringVar(&cfg.DBPath, "dbpath", cfg.DBPath, "path to the SQLite database file")
	flags.StringVar(&cfg.Port, "port", cfg.Port, "port to listen on")
	flags.BoolVar(&cfg.AllowRemoteWrites, "allow-remote-writes", cfg.AllowRemoteWrites, "allow remote gRPC clients to modify data")
	flags.BoolVar(&cfg.TLS.Enabled, "server-enable-mtls", cfg.TLS.Enabled, "require mutual TLS for server connections")
	flags.StringVar(&cfg.TLS.CertFile, "server-cert", cfg.TLS.CertFile, "path to the server certificate")
	flags.StringVar(&cfg.TLS.KeyFile, "server-key", cfg.TLS.KeyFile, "path to the server private key")
	flags.StringVar(&cfg.TLS.CAFile, "server-ca-cert", cfg.TLS.CAFile, "path to the CA used to verify clients")
	flags.IntVar(&cfg.MaxRecvMsgSize, "max-recv-message-size", cfg.MaxRecvMsgSize, "maximum received gRPC message size in bytes")
	flags.IntVar(&cfg.MaxSendMsgSize, "max-send-message-size", cfg.MaxSendMsgSize, "maximum sent gRPC message size in bytes")
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}
	if flags.NArg() != 0 {
		return Config{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks values that would otherwise fail late during startup.
func (c Config) Validate() error {
	if c.DBPath == "" {
		return errors.New("database path must not be empty")
	}
	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("port must be an integer between 1 and 65535: %q", c.Port)
	}
	if c.MaxRecvMsgSize <= 0 || c.MaxSendMsgSize <= 0 {
		return errors.New("gRPC message sizes must be positive")
	}
	if c.TLS.Enabled && (c.TLS.CertFile == "" || c.TLS.KeyFile == "" || c.TLS.CAFile == "") {
		return errors.New("server mTLS requires --server-cert, --server-key, and --server-ca-cert")
	}
	return nil
}

// LoadServerTLS loads mutual TLS credentials. It fails closed when TLS is
// enabled but credentials are invalid.
func (c TLSConfig) LoadServerTLS() (*tls.Config, error) {
	if !c.Enabled {
		return nil, nil
	}
	certificate, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load server certificate and key: %w", err)
	}
	caPEM, err := os.ReadFile(c.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read client CA certificate: %w", err)
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("client CA certificate does not contain a valid PEM certificate")
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAs,
	}, nil
}

func stringEnv(lookup LookupEnv, name, fallback string) string {
	if value, ok := lookup(name); ok {
		return value
	}
	return fallback
}

func boolEnv(lookup LookupEnv, name string, fallback bool) (bool, error) {
	value, ok := lookup(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", name, err)
	}
	return parsed, nil
}
