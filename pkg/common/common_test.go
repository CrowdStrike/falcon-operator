package common

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/version"
	ktest "k8s.io/client-go/testing"
)

type FakeDiscovery struct {
	*ktest.Fake
	FakedServerVersion *version.Info
}

func TestInitContainerClusterIDArgs(t *testing.T) {
	want := []string{"-c", `set -e; if [ ! -f /opt/CrowdStrike/falcon-daemonset-init ]; then echo "Error: This is not a falcon node sensor(DaemonSet) image"; exit 1; fi; echo "Running /opt/CrowdStrike/falcon-daemonset-init -i"; /opt/CrowdStrike/falcon-daemonset-init -i; if [ ! -f /opt/CrowdStrike/configure-cluster-id ]; then echo "/opt/CrowdStrike/configure-cluster-id not found. Skipping."; else echo "Running /opt/CrowdStrike/configure-cluster-id"; /opt/CrowdStrike/configure-cluster-id; fi`}
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

func TestCleanupSleep(t *testing.T) {
	want := []string{"-c", "sleep infinity"}
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

	if got := MakeSensorEnvMap(falconSensor.FalconSensor); !reflect.DeepEqual(got, sensorConfig) {
		t.Errorf("MakeSensorEnvMap() = %v, want %v", got, sensorConfig)
	}

	// Test getting proxy when APH and APP are set manually
	falconSensor.APH = "testmanually"
	falconSensor.APP = &port

	sensorConfig["FALCONCTL_OPT_APH"] = falconSensor.APH
	sensorConfig["FALCONCTL_OPT_APP"] = strconv.Itoa(*falconSensor.APP)

	if got := MakeSensorEnvMap(falconSensor.FalconSensor); !reflect.DeepEqual(got, sensorConfig) {
		t.Errorf("MakeSensorEnvMap() = %v, want %v", got, sensorConfig)
	}
}

func TestAppendUniqueEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		envVars  [][]corev1.EnvVar
		expected []corev1.EnvVar
	}{
		{
			name: "append to empty list",
			envVars: [][]corev1.EnvVar{
				{},
				{
					{Name: "VAR1", Value: "value1"},
					{Name: "VAR2", Value: "value2"},
				},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
			},
		},
		{
			name: "append with no duplicates",
			envVars: [][]corev1.EnvVar{
				{
					{Name: "VAR1", Value: "value1"},
				},
				{
					{Name: "VAR2", Value: "value2"},
					{Name: "VAR3", Value: "value3"},
				},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
				{Name: "VAR3", Value: "value3"},
			},
		},
		{
			name: "append with duplicates (should skip duplicates)",
			envVars: [][]corev1.EnvVar{
				{
					{Name: "VAR1", Value: "value1"},
					{Name: "VAR2", Value: "value2"},
				},
				{
					{Name: "VAR2", Value: "different_value"},
					{Name: "VAR3", Value: "value3"},
				},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
				{Name: "VAR3", Value: "value3"},
			},
		},
		{
			name: "append nil slice",
			envVars: [][]corev1.EnvVar{
				{
					{Name: "VAR1", Value: "value1"},
				},
				nil,
				{
					{Name: "VAR2", Value: "value2"},
				},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
			},
		},
		{
			name: "append multiple lists",
			envVars: [][]corev1.EnvVar{
				{
					{Name: "VAR1", Value: "value1"},
				},
				{
					{Name: "VAR2", Value: "value2"},
				},
				{
					{Name: "VAR3", Value: "value3"},
				},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
				{Name: "VAR3", Value: "value3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppendUniqueEnvVars(tt.envVars...)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("AppendUniqueEnvVars() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUpdateEnvVars(t *testing.T) {
	tests := []struct {
		name          string
		envVars       []corev1.EnvVar
		updateEnvVars []corev1.EnvVar
		expected      []corev1.EnvVar
	}{
		{
			name: "update existing var",
			envVars: []corev1.EnvVar{
				{Name: "VAR1", Value: "old_value"},
				{Name: "VAR2", Value: "value2"},
			},
			updateEnvVars: []corev1.EnvVar{
				{Name: "VAR1", Value: "new_value"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "new_value"},
				{Name: "VAR2", Value: "value2"},
			},
		},
		{
			name: "update multiple vars",
			envVars: []corev1.EnvVar{
				{Name: "VAR1", Value: "old_value1"},
				{Name: "VAR2", Value: "old_value2"},
				{Name: "VAR3", Value: "value3"},
			},
			updateEnvVars: []corev1.EnvVar{
				{Name: "VAR1", Value: "new_value1"},
				{Name: "VAR2", Value: "new_value2"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "new_value1"},
				{Name: "VAR2", Value: "new_value2"},
				{Name: "VAR3", Value: "value3"},
			},
		},
		{
			name: "update non-existent var (should not add)",
			envVars: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			updateEnvVars: []corev1.EnvVar{
				{Name: "VAR2", Value: "value2"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
		},
		{
			name: "update with same value (no change)",
			envVars: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			updateEnvVars: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
		},
		{
			name:    "update empty list",
			envVars: []corev1.EnvVar{},
			updateEnvVars: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			expected: []corev1.EnvVar{},
		},
		{
			name: "empty update list",
			envVars: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			updateEnvVars: []corev1.EnvVar{},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpdateEnvVars(tt.envVars, tt.updateEnvVars)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("UpdateEnvVars() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMergeEnvVars(t *testing.T) {
	tests := []struct {
		name           string
		envA           []corev1.EnvVar
		envB           []corev1.EnvVar
		envVarsToMerge []string
		expected       []corev1.EnvVar
	}{
		{
			name: "merge proxy vars from B to A",
			envA: []corev1.EnvVar{
				{Name: "APP_MODE", Value: "production"},
				{Name: "HTTP_PROXY", Value: "old-proxy:8080"},
				{Name: "DATABASE", Value: "postgres"},
			},
			envB: []corev1.EnvVar{
				{Name: "HTTP_PROXY", Value: "new-proxy:9090"},
				{Name: "HTTPS_PROXY", Value: "new-proxy:9443"},
			},
			envVarsToMerge: []string{"HTTP_PROXY", "HTTPS_PROXY"},
			expected: []corev1.EnvVar{
				{Name: "APP_MODE", Value: "production"},
				{Name: "DATABASE", Value: "postgres"},
				{Name: "HTTP_PROXY", Value: "new-proxy:9090"},
				{Name: "HTTPS_PROXY", Value: "new-proxy:9443"},
			},
		},
		{
			name: "merge non-existent var from B (should skip)",
			envA: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			envB: []corev1.EnvVar{
				{Name: "VAR2", Value: "value2"},
			},
			envVarsToMerge: []string{"VAR3"},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
		},
		{
			name: "empty merge list returns envA unchanged",
			envA: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			envB: []corev1.EnvVar{
				{Name: "VAR2", Value: "value2"},
			},
			envVarsToMerge: []string{},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
		},
		{
			name: "nil merge list returns envA unchanged",
			envA: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			envB: []corev1.EnvVar{
				{Name: "VAR2", Value: "value2"},
			},
			envVarsToMerge: nil,
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
		},
		{
			name: "merge all vars",
			envA: []corev1.EnvVar{
				{Name: "VAR1", Value: "old1"},
				{Name: "VAR2", Value: "old2"},
			},
			envB: []corev1.EnvVar{
				{Name: "VAR1", Value: "new1"},
				{Name: "VAR2", Value: "new2"},
			},
			envVarsToMerge: []string{"VAR1", "VAR2"},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "new1"},
				{Name: "VAR2", Value: "new2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeEnvVars(tt.envA, tt.envB, tt.envVarsToMerge)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("MergeEnvVars() = %v, want %v", got, tt.expected)
			}
		})
	}
}
