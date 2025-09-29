package e2e

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/crowdstrike/falcon-operator/test/utils"
	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
)

// Toleration represents the expected toleration to validate
type ExpectedToleration struct {
	Key      string
	Operator string
	Value    string
	Effect   string
}

// validateDeploymentTolerations checks if a deployment has the expected tolerations
func (cr crConfig) validateDeploymentTolerations(deploymentName string, expectedTolerations []ExpectedToleration) {
	By(fmt.Sprintf("validating tolerations for deployment %s in namespace %s", deploymentName, cr.namespace))

	checkTolerations := func() error {
		// Get the deployment as JSON
		cmd := exec.Command("kubectl", "get", "deployment", deploymentName,
			"-n", cr.namespace, "-o", "json")
		output, err := utils.Run(cmd)
		if err != nil {
			return fmt.Errorf("failed to get deployment: %v", err)
		}

		// Parse the deployment
		var deployment appsv1.Deployment
		err = json.Unmarshal(output, &deployment)
		if err != nil {
			return fmt.Errorf("failed to parse deployment JSON: %v", err)
		}

		// Get the tolerations from the pod spec
		podTolerations := deployment.Spec.Template.Spec.Tolerations

		// Verify each expected toleration exists
		for _, expected := range expectedTolerations {
			found := false
			for _, toleration := range podTolerations {
				if matchesToleration(toleration, expected) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("expected toleration not found: Key=%s, Operator=%s, Value=%s, Effect=%s",
					expected.Key, expected.Operator, expected.Value, expected.Effect)
			}
		}

		fmt.Printf("Found %d tolerations in deployment %s\n", len(podTolerations), deploymentName)
		for _, t := range podTolerations {
			fmt.Printf("  - Key: %s, Operator: %s, Value: %s, Effect: %s\n",
				t.Key, t.Operator, t.Value, t.Effect)
		}

		return nil
	}

	EventuallyWithOffset(1, checkTolerations, defaultTimeout, defaultPollPeriod).Should(Succeed())
}

// matchesToleration checks if a toleration matches the expected values
func matchesToleration(actual corev1.Toleration, expected ExpectedToleration) bool {
	// Handle empty operator (defaults to Equal)
	actualOperator := string(actual.Operator)
	if actualOperator == "" {
		actualOperator = "Equal"
	}
	expectedOperator := expected.Operator
	if expectedOperator == "" {
		expectedOperator = "Equal"
	}

	return actual.Key == expected.Key &&
		actualOperator == expectedOperator &&
		actual.Value == expected.Value &&
		string(actual.Effect) == expected.Effect
}

// validateDaemonSetTolerations checks if a daemonset has the expected tolerations
func (cr crConfig) validateDaemonSetTolerations(daemonsetName string, expectedTolerations []ExpectedToleration) {
	By(fmt.Sprintf("validating tolerations for daemonset %s in namespace %s", daemonsetName, cr.namespace))

	checkTolerations := func() error {
		// Get the daemonset as JSON
		cmd := exec.Command("kubectl", "get", "daemonset", daemonsetName,
			"-n", cr.namespace, "-o", "json")
		output, err := utils.Run(cmd)
		if err != nil {
			return fmt.Errorf("failed to get daemonset: %v", err)
		}

		// Parse the daemonset
		var daemonset appsv1.DaemonSet
		err = json.Unmarshal(output, &daemonset)
		if err != nil {
			return fmt.Errorf("failed to parse daemonset JSON: %v", err)
		}

		// Get the tolerations from the pod spec
		podTolerations := daemonset.Spec.Template.Spec.Tolerations

		// Verify each expected toleration exists
		for _, expected := range expectedTolerations {
			found := false
			for _, toleration := range podTolerations {
				if matchesToleration(toleration, expected) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("expected toleration not found: Key=%s, Operator=%s, Value=%s, Effect=%s",
					expected.Key, expected.Operator, expected.Value, expected.Effect)
			}
		}

		fmt.Printf("Found %d tolerations in daemonset %s\n", len(podTolerations), daemonsetName)
		for _, t := range podTolerations {
			fmt.Printf("  - Key: %s, Operator: %s, Value: %s, Effect: %s\n",
				t.Key, t.Operator, t.Value, t.Effect)
		}

		return nil
	}

	EventuallyWithOffset(1, checkTolerations, defaultTimeout, defaultPollPeriod).Should(Succeed())
}

// getDeploymentName retrieves the deployment name for a given CR
func (cr crConfig) getDeploymentName() (string, error) {
	cmd := exec.Command("kubectl", "get", "deployments",
		"-n", cr.namespace, "-o", "jsonpath={.items[0].metadata.name}")
	output, err := utils.Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to get deployment name: %v", err)
	}

	deploymentName := strings.TrimSpace(string(output))
	if deploymentName == "" {
		return "", fmt.Errorf("no deployment found in namespace %s", cr.namespace)
	}

	return deploymentName, nil
}