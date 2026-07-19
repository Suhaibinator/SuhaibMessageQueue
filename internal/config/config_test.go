package config

import "testing"

func TestLoadPrecedence(t *testing.T) {
	env := map[string]string{
		EnvDBPath:            "env.db",
		EnvPort:              "9000",
		EnvAllowRemoteWrites: "false",
	}
	lookup := func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	}

	cfg, err := Load([]string{"--dbpath=flag.db", "--port=9001", "--allow-remote-writes=true"}, lookup)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBPath != "flag.db" || cfg.Port != "9001" || !cfg.AllowRemoteWrites {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadRejectsInvalidBoolean(t *testing.T) {
	lookup := func(key string) (string, bool) {
		if key == EnvAllowRemoteWrites {
			return "sometimes", true
		}
		return "", false
	}
	if _, err := Load(nil, lookup); err == nil {
		t.Fatal("Load() expected an error")
	}
}

func TestValidateRequiresCompleteTLSConfiguration(t *testing.T) {
	cfg := Config{
		DBPath:         "smq.db",
		Port:           "8097",
		MaxRecvMsgSize: 1,
		MaxSendMsgSize: 1,
		TLS:            TLSConfig{Enabled: true},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() expected an error")
	}
}

func TestValidateRejectsInvalidPort(t *testing.T) {
	cfg := Config{DBPath: "smq.db", Port: "70000", MaxRecvMsgSize: 1, MaxSendMsgSize: 1}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() expected an error")
	}
}
