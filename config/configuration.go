package config

const (
	ENV_DB_PATH     = "DB_PATH"
	ENV_PORT        = "PORT"
	ENV_CLIENT_CERT = "CLIENT_CERT"
	ENV_CLIENT_KEY  = "CLIENT_KEY"
	ENV_CA_CERT     = "CA_CERT"
	ENV_ENABLE_MTLS = "ENABLE_MTLS" // Client-side mTLS

	// Server-side mTLS environment variables
	ENV_SERVER_ENABLE_MTLS  = "SERVER_ENABLE_MTLS"
	ENV_SERVER_CERT_FILE    = "SERVER_CERT_FILE"
	ENV_SERVER_KEY_FILE     = "SERVER_KEY_FILE"
	ENV_SERVER_CA_CERT_FILE = "SERVER_CA_CERT_FILE"

	// Control whether remote gRPC clients can modify data
	ENV_ALLOW_REMOTE_WRITES = "ALLOW_REMOTE_WRITES"
)

var (
	// DBPath is the path to the SQLite database file
	DBPath = "./dbtest.db"
	Port   = "8097"

	// Client mTLS configuration
	ClientCert = ""    // Path to client certificate file
	ClientKey  = ""    // Path to client key file
	CACert     = ""    // Path to CA certificate file for verifying the server
	EnableMTLS = false // Flag to enable/disable client-side mTLS

	// Server mTLS configuration
	ServerEnableMTLS = false // Flag to enable/disable server-side mTLS
	ServerCertFile   = ""    // Path to server's certificate file
	ServerKeyFile    = ""    // Path to server's private key file
	ServerCACertFile = ""    // Path to CA certificate file for verifying client certificates

	// AllowRemoteWrites controls whether external gRPC callers can perform write operations
	AllowRemoteWrites = true
)
