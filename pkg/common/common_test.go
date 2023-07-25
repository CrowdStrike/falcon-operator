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
	want := []string{"-c", `if [ -x "/opt/CrowdStrike/falcon-daemonset-init" ]; then echo "Executing falcon-daemonset-init -i"; falcon-daemonset-init -i ; else if [ -d "/host_opt/CrowdStrike/falconstore" ]; then echo "Re-creating /opt/CrowdStrike/falconstore as it is a directory instead of a file"; rm -rf /host_opt/CrowdStrike/falconstore; fi; mkdir -p /host_opt/CrowdStrike/ && touch /host_opt/CrowdStrike/falconstore; fi`}
	if got := InitContainerArgs(); !reflect.DeepEqual(got, want) {
		t.Errorf("InitContainerArgs() = %v, want %v", got, want)
	}
}

func TestInitCleanupArgs(t *testing.T) {
	want := []string{"-c", `if [ -x "/opt/CrowdStrike/falcon-daemonset-init" ]; then echo "Running falcon-daemonset-init -u"; falcon-daemonset-init -u; else echo "Manually removing /host_opt/CrowdStrike/"; rm -rf /host_opt/CrowdStrike/; fi`}
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

func TestGetProxyInfo(t *testing.T) {
	os.Setenv("HTTPS_PROXY", "https://test:1234")
	host, port := "https://test", "1234"
	gotHost, gotPort := getProxyInfo()
	if !reflect.DeepEqual(gotHost, host) {
		t.Errorf("getProxyInfo() = host: %v, want host: %v", gotHost, host)
	}
	if !reflect.DeepEqual(gotPort, port) {
		t.Errorf("getProxyInfo() = port: %v, want port: %v", gotPort, port)
	}

	os.Setenv("HTTP_PROXY", "http://user:password@test:1234")
	host, port = "http://user:password@test", "1234"
	gotHost, gotPort = getProxyInfo()
	if !reflect.DeepEqual(gotHost, host) {
		t.Errorf("getProxyInfo() = host: %v, want host: %v", gotHost, host)
	}
	if !reflect.DeepEqual(gotPort, port) {
		t.Errorf("getProxyInfo() = port: %v, want port: %v", gotPort, port)
	}

	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("HTTP_PROXY")
	host, port = "", ""
	gotHost, gotPort = getProxyInfo()
	if !reflect.DeepEqual(gotHost, host) {
		t.Errorf("getProxyInfo() = host: %v, want host: %v", gotHost, host)
	}
	if !reflect.DeepEqual(gotPort, port) {
		t.Errorf("getProxyInfo() = port: %v, want port: %v", gotPort, port)
	}
}

func TestMakeSensorEnvMap(t *testing.T) {
	falconNode := falconv1alpha1.FalconNodeSensor{}
	falconSensor := falconNode.Spec.Falcon
	sensorConfig := make(map[string]string)
	proxy := NewProxyInfo()
	port := 1234
	pDisabled := false
	cid := "test"

	falconSensor.APH = "test"
	falconSensor.APP = &port
	falconSensor.APD = &pDisabled
	falconSensor.Billing = "test"
	falconSensor.CID = &cid
	falconSensor.PToken = "test"
	falconSensor.Tags = []string{"test"}
	falconSensor.Trace = "debug"

	os.Setenv("HTTPS_PROXY", "https://test.proxy:666")

	// Set proxy values from environment variables if they exist
	if proxy.Host() != "" {
		sensorConfig["FALCONCTL_OPT_APH"] = proxy.Host()
	}
	if proxy.Port() != "" {
		sensorConfig["FALCONCTL_OPT_APP"] = proxy.Port()
	}

	// Set sensor values from CRD
	if falconSensor.APD != nil {
		sensorConfig["FALCONCTL_OPT_APD"] = strconv.FormatBool(*falconSensor.APD)
	}
	if falconSensor.APH != "" {
		sensorConfig["FALCONCTL_OPT_APH"] = falconSensor.APH
	}
	if falconSensor.APP != nil {
		sensorConfig["FALCONCTL_OPT_APP"] = strconv.Itoa(*falconSensor.APP)
	}
	if falconSensor.Billing != "" {
		sensorConfig["FALCONCTL_OPT_BILLING"] = falconSensor.Billing
	}
	if falconSensor.PToken != "" {
		sensorConfig["FALCONCTL_OPT_PROVISIONING_TOKEN"] = falconSensor.PToken
	}
	if len(falconSensor.Tags) > 0 {
		sensorConfig["FALCONCTL_OPT_TAGS"] = strings.Join(falconSensor.Tags, ",")
	}
	if falconSensor.Trace != "" {
		sensorConfig["FALCONCTL_OPT_TRACE"] = falconSensor.Trace
	}

	if got := MakeSensorEnvMap(falconSensor); !reflect.DeepEqual(got, sensorConfig) {
		t.Errorf("MakeSensorEnvMap() = %v, want %v", got, sensorConfig)
	}
}
