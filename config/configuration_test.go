package config

import (
	"flag"
	"os"
	"testing"
)

func TestDefaultConfiguration(t *testing.T) {
	// Save original values
	originalDBPath := DBPath
	originalPort := Port

	// Reset to defaults after test
	defer func() {
		DBPath = originalDBPath
		Port = originalPort
	}()

	// Reset to default values for test
	DBPath = "./dbtest.db"
	Port = "8097"

	// Verify default values
	if DBPath != "./dbtest.db" {
		t.Errorf("Expected default DBPath to be './dbtest.db', got '%s'", DBPath)
	}

	if Port != "8097" {
		t.Errorf("Expected default Port to be '8097', got '%s'", Port)
	}
}

func TestEnvironmentVariableOverrides(t *testing.T) {
	// Save original values
	originalDBPath := DBPath
	originalPort := Port

	// Reset to defaults after test
	defer func() {
		DBPath = originalDBPath
		Port = originalPort
		os.Unsetenv(ENV_DB_PATH)
		os.Unsetenv(ENV_PORT)
	}()

	// Reset to default values for test
	DBPath = "./dbtest.db"
	Port = "8097"

	// Set environment variables
	os.Setenv(ENV_DB_PATH, "./env_test.db")
	os.Setenv(ENV_PORT, "9000")

	// Check environment variables
	if dbPath, exists := os.LookupEnv(ENV_DB_PATH); exists {
		DBPath = dbPath
	}
	if port, exists := os.LookupEnv(ENV_PORT); exists {
		Port = port
	}

	// Verify values were overridden
	if DBPath != "./env_test.db" {
		t.Errorf("Expected DBPath to be './env_test.db', got '%s'", DBPath)
	}

	if Port != "9000" {
		t.Errorf("Expected Port to be '9000', got '%s'", Port)
	}
}

func TestCommandLineFlagOverrides(t *testing.T) {
	// Save original values
	originalDBPath := DBPath
	originalPort := Port

	// Save original command line arguments
	oldArgs := os.Args

	// Reset to defaults after test
	defer func() {
		DBPath = originalDBPath
		Port = originalPort
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	// Reset to default values for test
	DBPath = "./dbtest.db"
	Port = "8097"

	// Set up command line arguments
	os.Args = []string{"cmd", "-dbpath=./flag_test.db", "-port=9001"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Parse flags
	flag.StringVar(&DBPath, "dbpath", DBPath, "path to the SQLite database file")
	flag.StringVar(&Port, "port", Port, "port to listen on")
	flag.Parse()

	// Verify values were overridden
	if DBPath != "./flag_test.db" {
		t.Errorf("Expected DBPath to be './flag_test.db', got '%s'", DBPath)
	}

	if Port != "9001" {
		t.Errorf("Expected Port to be '9001', got '%s'", Port)
	}
}

func TestEnvironmentVariablePrecedence(t *testing.T) {
	// Save original values
	originalDBPath := DBPath
	originalPort := Port

	// Save original command line arguments
	oldArgs := os.Args

	// Reset to defaults after test
	defer func() {
		DBPath = originalDBPath
		Port = originalPort
		os.Args = oldArgs
		os.Unsetenv(ENV_DB_PATH)
		os.Unsetenv(ENV_PORT)
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	// Reset to default values for test
	DBPath = "./dbtest.db"
	Port = "8097"

	// Set environment variables
	os.Setenv(ENV_DB_PATH, "./env_test.db")
	os.Setenv(ENV_PORT, "9000")

	// Set up command line arguments
	os.Args = []string{"cmd", "-dbpath=./flag_test.db", "-port=9001"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// First check environment variables
	if dbPath, exists := os.LookupEnv(ENV_DB_PATH); exists {
		DBPath = dbPath
	}
	if port, exists := os.LookupEnv(ENV_PORT); exists {
		Port = port
	}

	// Then parse flags (which should override environment variables)
	flag.StringVar(&DBPath, "dbpath", DBPath, "path to the SQLite database file")
	flag.StringVar(&Port, "port", Port, "port to listen on")
	flag.Parse()

	// Verify command line flags take precedence over environment variables
	if DBPath != "./flag_test.db" {
		t.Errorf("Expected DBPath to be './flag_test.db', got '%s'", DBPath)
	}

	if Port != "9001" {
		t.Errorf("Expected Port to be '9001', got '%s'", Port)
	}
}
