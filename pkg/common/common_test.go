package common

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/version"
	ktest "k8s.io/client-go/testing"
)

type FakeDiscovery struct {
	*ktest.Fake
	FakedServerVersion *version.Info
}

func TestInitContainerArgs(t *testing.T) {
	want := []string{"-c", `if [ -x "/opt/CrowdStrike/falcon-daemonset-init" ]; then echo "Executing falcon-daemonset-init -i"; falcon-daemonset-init -i ; else if [ -d "/host_opt/CrowdStrike/falconstore" ]; then echo "Re-creating /opt/CrowdStrike/falconstore as it is a directory instead of a file"; rm -rf /host_opt/CrowdStrike/falconstore; fi; mkdir -p /host_opt/CrowdStrike/ && touch /host_opt/CrowdStrike/falconstore; fi`}
	if got := InitContainerArgs(); !reflect.DeepEqual(got, want) {
		t.Errorf("InitContainerArgs() = %v, want %v", got, want)
	}
}

func TestInitCleanupArgs(t *testing.T) {
	want := []string{"-c", `if [ -x "/opt/CrowdStrike/falcon-daemonset-init" ]; then echo "Running falcon-daemonset-init -u"; falcon-daemonset-init -u; else echo "Manually removing /host_opt/CrowdStrike"; rm -rf /host_opt/CrowdStrike; fi`}
	if got := InitCleanupArgs(); !reflect.DeepEqual(got, want) {
		t.Errorf("InitCleanupArgs() = %v, want %v", got, want)
	}
}

func TestCleanupSleep(t *testing.T) {
	want := []string{"-c", "sleep 10"}
	if got := CleanupSleep(); !reflect.DeepEqual(got, want) {
		t.Errorf("CleanupSleep() = %v, want %v", got, want)
	}
}

func TestEncodedBase64String(t *testing.T) {
	want := "dGVzdA=="
	if got := string(EncodedBase64String("test")); !reflect.DeepEqual(got, want) {
		t.Errorf("EncodedBase64String() = %v, want %v", got, want)
	}
}

func TestCleanDecodedBase64(t *testing.T) {
	want := []byte("test")

	if got := CleanDecodedBase64(EncodedBase64String("\t\ntest\n\t")); !reflect.DeepEqual(got, want) {
		t.Errorf("CleanDecodedBase64() = %v, want %v", got, want)
	}
}
