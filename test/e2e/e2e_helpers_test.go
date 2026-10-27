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
func validateNoReconcileLoop(controllerPodName, namespace, kind string, duration time.Duration, threshold int) {
	By(fmt.Sprintf("validating no infinite reconcile loop for %s over %v (threshold: %d)", kind, duration, threshold))

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

	if reconcileCount > threshold {
		Fail(fmt.Sprintf("Infinite reconcile loop detected for %s: %d unique reconcile operations in %v (expected: <= %d)",
			kind, reconcileCount, duration, threshold))
	}
}

// operatorDeploymentName returns the name of the operator Deployment in the operator namespace
func operatorDeploymentName() string {
	cmd := exec.Command("kubectl", "get", "deployment", "-l", "control-plane=controller-manager",
		"-n", namespace, "-o", "jsonpath={.items[0].metadata.name}")
	output, err := utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	name := strings.TrimSpace(string(output))
	ExpectWithOffset(1, name).NotTo(BeEmpty(), "operator Deployment not found in namespace %s", namespace)
	return name
}

// isOperatorManagedByOLM reports whether OLM owns the operator Deployment. OLM reverts direct changes
// to the Deployment, so env vars cannot be changed with kubectl set env.
func isOperatorManagedByOLM(deployment string) bool {
	cmd := exec.Command("kubectl", "get", "deployment", deployment, "-n", namespace,
		"-o", "jsonpath={.metadata.labels.olm\\.owner}")
	output, err := utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return strings.TrimSpace(string(output)) != ""
}

// getOperatorEnv returns the value of an env var on the operator manager container, or "" if it is not set
func getOperatorEnv(deployment, name string) string {
	cmd := exec.Command("kubectl", "get", "deployment", deployment, "-n", namespace,
		"-o", fmt.Sprintf("jsonpath={.spec.template.spec.containers[?(@.name==\"manager\")].env[?(@.name==\"%s\")].value}", name))
	output, err := utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return strings.TrimSpace(string(output))
}

// setOperatorEnv sets an env var on the operator manager container, or removes it when value is "",
// then waits for the new operator pod to be running and updates controllerPodName
func setOperatorEnv(deployment, name, value string) {
	envArg := fmt.Sprintf("%s=%s", name, value)
	if value == "" {
		envArg = name + "-"
	}

	By(fmt.Sprintf("setting %s on the operator Deployment", envArg))
	cmd := exec.Command("kubectl", "set", "env", "deployment/"+deployment, "-n", namespace, "-c", "manager", envArg)
	_, err := utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	cmd = exec.Command("kubectl", "rollout", "status", "deployment/"+deployment, "-n", namespace, "--timeout=180s")
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	waitForControllerPod()
}

// waitForControllerPod waits for a single running operator pod and stores its name in controllerPodName
func waitForControllerPod() {
	getControllerPod := func(g Gomega) {
		cmd := exec.Command("kubectl", "get",
			"pods", "-l", "control-plane=controller-manager",
			"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
				"{{ \"\\n\" }}{{ end }}{{ end }}",
			"-n", namespace,
		)
		podOutput, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		podNames := utils.GetNonEmptyLines(string(podOutput))
		g.Expect(podNames).To(HaveLen(1))

		cmd = exec.Command("kubectl", "get", "pods", podNames[0], "-o", "jsonpath={.status.phase}", "-n", namespace)
		status, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(status)).To(Equal("Running"))

		controllerPodName = podNames[0]
	}
	EventuallyWithOffset(1, getControllerPod, defaultTimeout, defaultPollPeriod).Should(Succeed())
}

// validateOperatorLogError waits until the operator logs contain expectedError and checks that the
// operator has not recovered from a panic while reconciling
func validateOperatorLogError(expectedError string) {
	By(fmt.Sprintf("validating that the operator logs %q without panicking", expectedError))
	validateLogs := func(g Gomega) {
		cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace, "-c", "manager", "--tail=-1")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(output)).NotTo(ContainSubstring("Observed a panic"))
		g.Expect(string(output)).To(ContainSubstring(expectedError))
	}
	EventuallyWithOffset(1, validateLogs, defaultTimeout, defaultPollPeriod).Should(Succeed())
}

// useBundledImage sets the RELATED_IMAGE_* env var on the operator, as the OpenShift OLM bundle does, and
// returns the image and a function that restores the env var. Under OLM, the bundle's value is used instead.
// Restore it from an AfterAll, not DeferCleanup, which runs after the suite's AfterAll undeploys the operator.
func useBundledImage(envVar, fakeImage string) (string, func()) {
	deployment := operatorDeploymentName()
	original := getOperatorEnv(deployment, envVar)

	if isOperatorManagedByOLM(deployment) {
		if original == "" {
			Skip(fmt.Sprintf("operator is managed by OLM and the bundle does not set %s", envVar))
		}
		return original, func() {}
	}

	setOperatorEnv(deployment, envVar, fakeImage)
	return fakeImage, func() {
		setOperatorEnv(deployment, envVar, original)
	}
}
