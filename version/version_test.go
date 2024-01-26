package version

import (
	"regexp"
	"testing"
)

func TestEmptyVersion(t *testing.T) {
	ver := Version
	if ver != "" {
		t.Error("Version returned value, expected only empty string. Got:", ver)
	}
}

func TestGoVersion(t *testing.T) {
	ver := GoVersion
	if ver == "" {
		t.Error("GoVersion returned empty string, expected value. Got:", ver)
	}

	returnVersion, err := regexp.MatchString("go[0-9]+.[0-9]+.[0-9]+.*", ver)
	if err != nil {
		t.Error("GoVersion returned error, expected value. Got:", err)
	}
	if !returnVersion {
		t.Error("GoVersion returned unexpected value. Got:", ver)
	}
}

func TestGet(t *testing.T) {
	ver := Get()
	if ver != "" {
		t.Error("Get returned value, expected only empty string. Got:", ver)
	}
}
