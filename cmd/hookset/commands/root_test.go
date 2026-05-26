package commands

import (
	"testing"
)

func TestBuildVersion(t *testing.T) {
	version := buildVersion()
	if version == "" {
		t.Error("buildVersion() should not return empty string")
	}
	// Version can be "dev" if not built with version info
}

func TestVerboseFlag(t *testing.T) {
	// Test default value
	if verbose {
		t.Error("verbose should default to false")
	}

	// Test setting it
	verbose = true
	defer func() { verbose = false }()

	if !verbose {
		t.Error("verbose should be true after setting")
	}
}
