package common

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"k8s.io/apimachinery/pkg/version"
	ktest "k8s.io/client-go/testing"
)

type FakeDiscovery struct {
	*ktest.Fake
	FakedServerVersion *version.Info
}

func TestInitContainerArgs(t *testing.T) {
	want := []string{"-c", `echo "Running /opt/CrowdStrike/falcon-daemonset-init -i"; /opt/CrowdStrike/falcon-daemonset-init -i`}
	if got := InitContainerArgs(); !reflect.DeepEqual(got, want) {
		t.Errorf("InitContainerArgs() = %v, want %v", got, want)
	}
}

func TestInitCleanupArgs(t *testing.T) {
	want := []string{"-c", `echo "Running /opt/CrowdStrike/falcon-daemonset-init -u"; /opt/CrowdStrike/falcon-daemonset-init -u`}
	if got := InitCleanupArgs(); !reflect.DeepEqual(got, want) {
		t.Errorf("InitCleanupArgs() = %v, want %v", got, want)
	}
}

func TestLegacyInitContainerArgs(t *testing.T) {
	want := []string{"-c", `if [ -x "/opt/CrowdStrike/falcon-daemonset-init -i" ]; then echo "Executing falcon-daemonset-init -i"; falcon-daemonset-init -i ; else if [ -d "/host_opt/CrowdStrike/falconstore" ]; then echo "Re-creating /opt/CrowdStrike/falconstore as it is a directory instead of a file"; rm -rf /host_opt/CrowdStrike/falconstore; fi; mkdir -p /host_opt/CrowdStrike/ && touch /host_opt/CrowdStrike/falconstore; fi`}
	if got := LegacyInitContainerArgs(); !reflect.DeepEqual(got, want) {
		t.Errorf("LegacyInitContainerArgs() = %v, want %v", got, want)
	}
}

func TestLegacyInitCleanupArgs(t *testing.T) {
	want := []string{"-c", `if [ -x "/opt/CrowdStrike/falcon-daemonset-init -u" ]; then echo "Running falcon-daemonset-init -u"; falcon-daemonset-init -u; else echo "Manually removing /host_opt/CrowdStrike/"; rm -rf /host_opt/CrowdStrike/; fi`}
	if got := LegacyInitCleanupArgs(); !reflect.DeepEqual(got, want) {
		t.Errorf("LegacyInitCleanupArgs() = %v, want %v", got, want)
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

func TestProxyHost(t *testing.T) {
	proxy := NewProxyInfo()
	proxy.host = "test"
	want := "test"

	if got := proxy.Host(); !reflect.DeepEqual(got, want) {
		t.Errorf("proxy.Host() = %v, want %v", got, want)
	}
}

func TestProxyPort(t *testing.T) {
	proxy := NewProxyInfo()
	proxy.port = "1234"
	want := "1234"

	if got := proxy.Port(); !reflect.DeepEqual(got, want) {
		t.Errorf("proxy.Port() = %v, want %v", got, want)
	}
}

func TestMakeSensorEnvMap(t *testing.T) {
	falconNode := falconv1alpha1.FalconNodeSensor{}
	falconSensor := falconNode.Spec.Falcon
	sensorConfig := make(map[string]string)
	port := 1234
	pDisabled := false
	cid := "test"

	falconSensor.APD = &pDisabled
	falconSensor.Billing = "test"
	falconSensor.CID = &cid
	falconSensor.PToken = "test"
	falconSensor.Tags = []string{"test"}
	falconSensor.Trace = "debug"

	os.Setenv("HTTPS_PROXY", "https://test-automatic.proxy:666")
	proxy := NewProxyInfo()

	// Test getting proxy from environment
	sensorConfig["FALCONCTL_OPT_APH"] = strings.TrimPrefix(proxy.Host(), "https://")
	sensorConfig["FALCONCTL_OPT_APH"] = strings.TrimPrefix(proxy.Host(), "http://")
	sensorConfig["FALCONCTL_OPT_APP"] = proxy.Port()

	// Set sensor values from CRD
	sensorConfig["FALCONCTL_OPT_APD"] = strconv.FormatBool(*falconSensor.APD)
	sensorConfig["FALCONCTL_OPT_BILLING"] = falconSensor.Billing
	sensorConfig["FALCONCTL_OPT_PROVISIONING_TOKEN"] = falconSensor.PToken
	sensorConfig["FALCONCTL_OPT_TAGS"] = strings.Join(falconSensor.Tags, ",")
	sensorConfig["FALCONCTL_OPT_TRACE"] = falconSensor.Trace

	if got := MakeSensorEnvMap(falconSensor); !reflect.DeepEqual(got, sensorConfig) {
		t.Errorf("MakeSensorEnvMap() = %v, want %v", got, sensorConfig)
	}

	// Test getting proxy when APH and APP are set manually
	falconSensor.APH = "testmanually"
	falconSensor.APP = &port

	sensorConfig["FALCONCTL_OPT_APH"] = falconSensor.APH
	sensorConfig["FALCONCTL_OPT_APP"] = strconv.Itoa(*falconSensor.APP)

	if got := MakeSensorEnvMap(falconSensor); !reflect.DeepEqual(got, sensorConfig) {
		t.Errorf("MakeSensorEnvMap() = %v, want %v", got, sensorConfig)
	}
}
