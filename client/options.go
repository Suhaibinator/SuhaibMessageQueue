package client

import (
	"errors"
	"time"
)

const (
	// HeartbeatOffset identifies keepalive responses on streaming consumers.
	HeartbeatOffset int64 = -1

	defaultRequestTimeout = 100 * time.Second
	defaultMessageSize    = 1024 * 1024 * 1024
)

// TLSConfig contains client certificate paths for mutual TLS.
type TLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

// Options configures a Client.
type Options struct {
	Address           string
	Port              string
	TLS               *TLSConfig
	RequestTimeout    time.Duration
	MaxSendMsgSize    int
	MaxReceiveMsgSize int
}

func (o Options) withDefaults() (Options, error) {
	if o.Address == "" {
		return Options{}, errors.New("client address must not be empty")
	}
	if o.Port == "" {
		return Options{}, errors.New("client port must not be empty")
	}
	if o.RequestTimeout <= 0 {
		o.RequestTimeout = defaultRequestTimeout
	}
	if o.MaxSendMsgSize <= 0 {
		o.MaxSendMsgSize = defaultMessageSize
	}
	if o.MaxReceiveMsgSize <= 0 {
		o.MaxReceiveMsgSize = defaultMessageSize
	}
	if o.TLS != nil && (o.TLS.CertFile == "" || o.TLS.KeyFile == "" || o.TLS.CAFile == "") {
		return Options{}, errors.New("client mTLS requires certificate, key, and CA files")
	}
	return o, nil
}
