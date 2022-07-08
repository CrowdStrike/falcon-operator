package common

import (
	"io/ioutil"
	"strings"
)

var (
	NodeSelector            = map[string]string{"kubernetes.io/os": "linux"}
	FalconShellCommand      = []string{"/bin/bash"}
	OrigDSConfVersion       = "0"
	FalconOperatorNamespace = "falcon-operator"
	FalconInjectorCommand   = []string{"injector"}
	Validity                = 3650
	CertAuth                = CAcert{}
	altDNSSuffixList        = []string{"svc", "svc.cluster.local"}
)

type CAcert struct {
	Cert string
	Key  string
}

type Certs struct {
	Cert string
	Key  string
}

func init() {
	nsBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err == nil {
		ns := strings.TrimSpace(string(nsBytes))
		FalconOperatorNamespace = ns
	}
}
