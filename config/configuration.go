package config

const (
	ENV_DB_PATH     = "DB_PATH"
	ENV_PORT        = "PORT"
	ENV_CLIENT_CERT = "CLIENT_CERT"
	ENV_CLIENT_KEY  = "CLIENT_KEY"
	ENV_CA_CERT     = "CA_CERT"
	ENV_ENABLE_MTLS = "ENABLE_MTLS"
)

var (
	// DBPath is the path to the SQLite database file
	DBPath = "./dbtest.db"
	Port   = "8097"

	// mTLS configuration
	ClientCert = ""    // Path to client certificate file
	ClientKey  = ""    // Path to client key file
	CACert     = ""    // Path to CA certificate file
	EnableMTLS = false // Flag to enable/disable mTLS
)
