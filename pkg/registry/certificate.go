package registry

import (
	"os"

	"github.com/go-logr/logr"
)

const (
	saCertDir  = "/var/run/secrets/kubernetes.io/serviceaccount/"
	saCertPath = saCertDir + "ca.crt"
)

func CADirPath(log logr.Logger) string {
	if caPathExists(log) {
		return saCertDir
	} else {
		return ""
	}
}

func caPathExists(log logr.Logger) bool {
	_, err := os.Stat(saCertPath)
	if err != nil && !os.IsNotExist(err) {
		log.Error(err, "Received error when trying to stat k8s certificate", "path", saCertPath)
	}
	return err == nil
}
