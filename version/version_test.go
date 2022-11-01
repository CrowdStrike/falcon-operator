package version

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
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

func TestPrint(t *testing.T) {
	var buffer bytes.Buffer

	opts := zap.Options{
		Development: true,
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts), zap.WriteTo(&buffer)))
	writer := bufio.NewWriter(&buffer)
	log = ctrl.Log.WithName("version")

	Print()
	writer.Flush()

	printOutput := buffer.String()

	if printOutput == "" {
		t.Error("Print returned empty string, expected value. Got:", printOutput)
	}

	if !strings.Contains(printOutput, "{\"version\": \"\"}") && !strings.Contains(printOutput, "{\"go-version\": \"go") {
		t.Error("Print returned unexpected value. Got:", printOutput)
	}
}
