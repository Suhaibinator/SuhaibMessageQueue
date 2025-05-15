# mTLS Examples for Suhaib Message Queue (SMQ)

This directory contains example code demonstrating how to use mTLS (mutual TLS) authentication with the Suhaib Message Queue client.

## What is mTLS?

Mutual TLS (mTLS) is a two-way authentication process where both the client and server verify each other's identity using X.509 certificates. This adds an extra layer of security beyond traditional TLS, which only authenticates the server to the client.

## Prerequisites for Using mTLS

Before using mTLS with SMQ, you need:

1. A Certificate Authority (CA) certificate
2. A client certificate signed by the CA
3. A client private key

## Examples

### Direct Configuration (`direct_config`)

This example demonstrates how to manually construct a TLSConfig and use it when creating a client.

#### Key Points:

- Manually creates a `config.TLSConfig` instance
- Sets certificate paths directly
- Verifies certificate files exist before attempting to connect

```go
// Create a TLSConfig if mTLS is enabled
var tlsConfig *config.TLSConfig
if enableMTLS {
    tlsConfig = &config.TLSConfig{
        CertFile: clientCertPath,
        KeyFile:  clientKeyPath,
        CAFile:   caCertPath,
    }
}

// Create a client with optional mTLS
smqClient, err := client.NewClient("localhost", "8097", tlsConfig)
```

### Global Configuration (`global_config`)

This example demonstrates how to use the built-in configuration system and helper functions for mTLS.

#### Key Points:

- Uses environment variables or command-line flags to configure mTLS
- Uses the `config.GetTLSConfig()` helper function to obtain the TLS configuration
- Automatically handles the decision to use mTLS based on the configuration

```go
// Get TLS config from global configuration
tlsConfig := config.GetTLSConfig()

// Create a client with optional mTLS
smqClient, err := client.NewClient("localhost", config.Port, tlsConfig)
```

## Running the Examples

To run either example without mTLS (insecure):

```bash
go run examples/direct_config/main.go
# or
go run examples/global_config/main.go
```

To run with mTLS enabled, you need to:

1. Generate the required certificates (see below)
2. Update the example code with the correct paths to your certificates
3. Set `enableMTLS = true` in the direct example or `config.EnableMTLS = true` in the global example

## Generating Certificates for Testing

Here's a quick guide to generate self-signed certificates for testing:

```bash
# Generate a CA key and certificate
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 365 -key ca.key -out ca.crt -subj "/CN=Test CA"

# Generate a client key
openssl genrsa -out client.key 2048

# Generate a certificate signing request (CSR)
openssl req -new -key client.key -out client.csr -subj "/CN=Test Client"

# Sign the client CSR with the CA
openssl x509 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt

# Clean up
rm client.csr ca.srl
```

## Environment Variables

The following environment variables can be used to configure mTLS:

- `CLIENT_CERT`: Path to the client certificate file
- `CLIENT_KEY`: Path to the client key file
- `CA_CERT`: Path to the CA certificate file
- `ENABLE_MTLS`: Set to "true" to enable mTLS

## Command-Line Flags

The following command-line flags can be used:

- `--client-cert`: Path to the client certificate file
- `--client-key`: Path to the client key file
- `--ca-cert`: Path to the CA certificate file
- `--enable-mtls`: Use to enable mTLS
