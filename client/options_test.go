package client

import (
	"testing"
	"time"
)

func TestOptionsWithDefaults(t *testing.T) {
	options, err := (Options{Address: "localhost", Port: "8097"}).withDefaults()
	if err != nil {
		t.Fatalf("withDefaults() error = %v", err)
	}
	if options.RequestTimeout != defaultRequestTimeout {
		t.Fatalf("RequestTimeout = %v, want %v", options.RequestTimeout, defaultRequestTimeout)
	}
	if options.MaxSendMsgSize != defaultMessageSize {
		t.Fatalf("MaxSendMsgSize = %d, want %d", options.MaxSendMsgSize, defaultMessageSize)
	}
	if options.MaxReceiveMsgSize != defaultMessageSize {
		t.Fatalf("MaxReceiveMsgSize = %d, want %d", options.MaxReceiveMsgSize, defaultMessageSize)
	}
}

func TestOptionsPreserveExplicitValues(t *testing.T) {
	wantTimeout := 5 * time.Second
	options, err := (Options{
		Address:           "localhost",
		Port:              "8097",
		RequestTimeout:    wantTimeout,
		MaxSendMsgSize:    1024,
		MaxReceiveMsgSize: 2048,
	}).withDefaults()
	if err != nil {
		t.Fatalf("withDefaults() error = %v", err)
	}
	if options.RequestTimeout != wantTimeout || options.MaxSendMsgSize != 1024 || options.MaxReceiveMsgSize != 2048 {
		t.Fatalf("explicit options were changed: %+v", options)
	}
}

func TestOptionsValidation(t *testing.T) {
	tests := []struct {
		name    string
		options Options
	}{
		{name: "missing address", options: Options{Port: "8097"}},
		{name: "missing port", options: Options{Address: "localhost"}},
		{
			name: "incomplete mTLS",
			options: Options{
				Address: "localhost",
				Port:    "8097",
				TLS:     &TLSConfig{CertFile: "client.pem"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.options.withDefaults(); err == nil {
				t.Fatal("withDefaults() expected an error")
			}
		})
	}
}
