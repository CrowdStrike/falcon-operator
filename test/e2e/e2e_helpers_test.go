package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"

	"github.com/crowdstrike/falcon-operator/test/utils"
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

// validateNoReconcileLoop checks that the controller is not stuck in an infinite reconcile loop
// kind parameter specifies the CRD kind to filter logs (e.g., "FalconNodeSensor", "FalconAdmission")
func validateNoReconcileLoop(controllerPodName, namespace, kind string, duration time.Duration) {
	By(fmt.Sprintf("validating no infinite reconcile loop for %s over %v", kind, duration))

	// Sleep for duration + 5 seconds buffer to ensure any in-progress reconciles complete
	bufferDuration := duration + (5 * time.Second)
	time.Sleep(bufferDuration)

	cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace,
		"--since", duration.String(), "--tail=-1")
	output, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())

	// Extract unique reconcileIDs from the logs
	reconcileIDs := make(map[string]bool)
	for line := range strings.SplitSeq(string(output), "\n") {
		if strings.Contains(line, "reconciling "+kind) && strings.Contains(line, "reconcileID") {
			parts := strings.Split(line, `"reconcileID": "`)
			if len(parts) >= 2 {
				uuidParts := strings.Split(parts[1], `"`)
				if len(uuidParts) >= 1 {
					reconcileID := uuidParts[0]
					reconcileIDs[reconcileID] = true
				}
			}
		}
	}

	reconcileCount := len(reconcileIDs)
	By(fmt.Sprintf("detected %d unique reconcile operations for %s in the last %v", reconcileCount, kind, duration))

	if reconcileCount > 0 {
		Fail(fmt.Sprintf("Infinite reconcile loop detected for %s: %d unique reconcile operations in %v (expected: 0)",
			kind, reconcileCount, duration))
	}
}
