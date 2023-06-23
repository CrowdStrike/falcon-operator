package common

import (
	"os"
	"strings"
)

var (
	NodeSelector            = map[string]string{"kubernetes.io/os": "linux"}
	FalconShellCommand      = []string{"/bin/bash"}
	OrigDSConfVersion       = "0"
	FalconOperatorNamespace = "falcon-operator"
	FalconInjectorCommand   = []string{"injector"}
)

func init() {
	nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err == nil {
		ns := strings.TrimSpace(string(nsBytes))
		FalconOperatorNamespace = ns
	}
}
