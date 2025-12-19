package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// getOperatorSDKPath returns the path to operator-sdk executable, following the same logic as the Makefile
// It first checks LOCALBIN (./bin), then falls back to system PATH
func getOperatorSDKPath() (string, error) {
	// Get current working directory to construct LOCALBIN path
	pwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check LOCALBIN first (equivalent to $(LOCALBIN)/operator-sdk)
	localBinPath := filepath.Join(pwd, "bin", "operator-sdk")
	if _, err := os.Stat(localBinPath); err == nil {
		return localBinPath, nil
	}

	// Fall back to system PATH (equivalent to $(shell which operator-sdk))
	systemPath, err := exec.LookPath("operator-sdk")
	if err != nil {
		return "", fmt.Errorf("operator-sdk not found in LOCALBIN (%s) or system PATH: %w", localBinPath, err)
	}

	return systemPath, nil
}

// isOpenShift detects if the current cluster is OpenShift by checking for OpenShift-specific resources
func isOpenShift() bool {
	// Check for OpenShift-specific API resources that indicate we're on OpenShift
	// This is a common pattern used in operator development
	cmd := exec.Command("kubectl", "api-resources", "--api-group=config.openshift.io")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// If we can find OpenShift config resources, we're on OpenShift
	return len(output) > 0 && strings.Contains(string(output), "clusterversions")
}
