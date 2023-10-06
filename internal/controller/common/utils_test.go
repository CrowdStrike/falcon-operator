package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOLogMessage(t *testing.T) {
	want := "test.test"
	got := oLogMessage("test", "test")

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("oLogMessage() mismatch (-want +got):\n%s", diff)
	}
}

func TestLogMessage(t *testing.T) {
	want := "test test test"
	got := logMessage("test", "test", "test")

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("logMessage() mismatch (-want +got):\n%s", diff)
	}
}
