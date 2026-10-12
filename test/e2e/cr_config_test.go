package e2e

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/crowdstrike/falcon-operator/test/utils"
	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

// crConfig holds the configuration parameters for each CRD.
// It defines the basic metadata and namespace information needed for installation.
// Ensure that fields are set to values that match the manifests found in the config/samples directory.
type crConfig struct {
	kind          string
	namespace     string // InstallNamespace for the Spec
	metadataName  string // The name of the resource expected in metadata.name
	componentName string // Component name in the metadata.labels
}
type crOperation struct {
	command string
	action  string
}

var (
	kacConfig = crConfig{
		kind:          "FalconAdmission",
		namespace:     "falcon-kac",
		metadataName:  "falcon-kac",
		componentName: "admission_controller",
	}
	iarConfig = crConfig{
		kind:          "FalconImageAnalyzer",
		namespace:     "falcon-iar",
		metadataName:  "falcon-image-analyzer",
		componentName: "falcon-imageanalyzer",
	}
	nodeConfig = crConfig{
		kind:          "FalconNodeSensor",
		namespace:     "falcon-system",
		metadataName:  "falcon-node-sensor",
		componentName: "kernel_sensor",
	}
	sidecarConfig = crConfig{
		kind:          "FalconContainer",
		namespace:     "falcon-system",
		metadataName:  "falcon-container-sensor",
		componentName: "container_sensor",
	}
	secretConfig = crConfig{
		namespace: falconSecretNamespace,
	}
	falconDeploymentConfig = crConfig{
		kind:         "FalconDeployment",
		namespace:    namespace,
		metadataName: "falcon-deployment",
	}
	projectDir, _ = utils.GetProjectDir()
	crApply       = crOperation{command: "apply", action: "creating"}
	crDelete      = crOperation{command: "delete", action: "deleting"}
)

func (cr crConfig) validateCrStatus() {
	By("validating that the status of the custom resource created is updated or not")
	getStatus := func() error {
		// Check Success condition
		cmd := exec.Command("kubectl", "get", strings.ToLower(cr.kind),
			cr.metadataName, "-o", "jsonpath={.status.conditions[?(@.type==\"Success\")].status}",
			"-n", cr.namespace,
		)
		status, err := utils.Run(cmd)
		fmt.Println("Success:", string(status))
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		if string(status) != "True" {
			return fmt.Errorf("Success condition status should be True, got: %s", status)
		}

		// Check resource-specific condition (DaemonSetReady or DeploymentReady)
		var conditionType string
		if cr.kind == "FalconNodeSensor" {
			conditionType = "DaemonSetReady"
		} else {
			conditionType = "DeploymentReady"
		}

		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind),
			cr.metadataName, "-o", fmt.Sprintf("jsonpath={.status.conditions[?(@.type==\"%s\")].status}", conditionType),
			"-n", cr.namespace,
		)
		status, err = utils.Run(cmd)
		fmt.Printf("%s: %s\n", conditionType, string(status))
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		if string(status) != "True" {
			return fmt.Errorf("%s condition status should be True, got: %s", conditionType, status)
		}

		// For DaemonSets with init containers, verify primary container is running
		if cr.kind == "FalconNodeSensor" {
			componentLabel := fmt.Sprintf("crowdstrike.com/component=%s", cr.componentName)
			cmd = exec.Command("kubectl", "get", "pods", "-n", cr.namespace,
				"-l", componentLabel,
				"-o", "jsonpath={.items[*].status.initContainerStatuses[*].state.terminated.reason}",
			)
			initStatus, err := utils.Run(cmd)
			ExpectWithOffset(2, err).NotTo(HaveOccurred())
			if !strings.Contains(string(initStatus), "Completed") {
				return fmt.Errorf("init container should be Completed, got: %s", initStatus)
			}

			cmd = exec.Command("kubectl", "get", "pods", "-n", cr.namespace,
				"-l", componentLabel,
				"-o", "jsonpath={.items[*].status.containerStatuses[?(@.name!=\"\")].ready}",
			)
			containerReady, err := utils.Run(cmd)
			ExpectWithOffset(2, err).NotTo(HaveOccurred())
			if !strings.Contains(string(containerReady), "true") {
				return fmt.Errorf("primary container should be ready, got: %s", containerReady)
			}
			fmt.Println("Primary container: ready")
		}

		return nil
	}
	Eventually(getStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())
}

func (cr crConfig) deleteNamespace() {
	By(fmt.Sprintf("deleting namespace %s", cr.namespace))
	deleteCmd := exec.Command("kubectl", "delete", "ns", cr.namespace)
	_, err := utils.Run(deleteCmd)
	ExpectWithOffset(2, err).NotTo(HaveOccurred())
}

func (cr crConfig) waitForNamespaceDeletion() {
	By(fmt.Sprintf("waiting for %s namespace to be fully deleted", cr.namespace))
	cmd := exec.Command("kubectl", "wait", "--for=delete",
		fmt.Sprintf("namespace/%s", cr.namespace),
		"--timeout=300s")
	_, err := utils.Run(cmd)
	ExpectWithOffset(2, err).NotTo(HaveOccurred())
}

func (cr crConfig) validateRunningStatus(running bool) {
	phase := "=Running"
	if !running {
		phase = "!=Running"
	}

	By("validating that pod(s) status.phase" + phase)
	componentLabel := fmt.Sprintf("crowdstrike.com/component=%s", cr.componentName)
	getFalconNodeSensorPodStatus := func() error {
		cmd := exec.Command("kubectl", "get",
			"pods", "-A", "-l", componentLabel, "--field-selector=status.phase=Running",
			"-o", "jsonpath={.items[*].status}", "-n", cr.namespace,
		)
		status, err := utils.Run(cmd)
		fmt.Println(string(status))
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		if (!running && len(status) > 0) || (running && !strings.Contains(string(status), "\"phase\":\"Running\"")) {
			return fmt.Errorf("%s pod in %s status", cr.metadataName, status)
		}
		return nil
	}
	EventuallyWithOffset(1, getFalconNodeSensorPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())
}

func (cr crConfig) manageCrdInstance(crCmd crOperation, manifest string) {
	By(fmt.Sprintf("%s an instance of the %s Operand(CR)", crCmd.action, cr.kind))
	EventuallyWithOffset(1, func() error {
		cmd := exec.Command("kubectl", crCmd.command, "-f", filepath.Join(projectDir,
			manifest), "-n", cr.namespace)
		_, err := utils.Run(cmd)
		return err
	}, defaultTimeout, defaultPollPeriod).Should(Succeed())
}

// validateOperatorEnvVars checks that running sensor pods expose OPERATOR_VERSION and
// OPERATOR_MANIFEST env vars with non-empty resolved values via the Downward API.
func (cr crConfig) validateOperatorEnvVars() {
	By(fmt.Sprintf("validating OPERATOR_VERSION and OPERATOR_MANIFEST env vars on %s pods", cr.kind))

	componentLabel := fmt.Sprintf("crowdstrike.com/component=%s", cr.componentName)

	getPodName := func() (string, error) {
		cmd := exec.Command("kubectl", "get", "pods",
			"-n", cr.namespace,
			"-l", componentLabel,
			"--field-selector=status.phase=Running",
			"-o", "jsonpath={.items[0].metadata.name}",
		)
		output, err := utils.Run(cmd)
		if err != nil {
			return "", err
		}
		name := strings.TrimSpace(string(output))
		if name == "" {
			return "", fmt.Errorf("no running pods found with label %s", componentLabel)
		}
		return name, nil
	}

	validateEnvInPod := func(g Gomega) {
		podName, err := getPodName()
		g.Expect(err).NotTo(HaveOccurred())

		for _, envVar := range []string{"OPERATOR_VERSION", "OPERATOR_MANIFEST"} {
			cmd := exec.Command("kubectl", "exec", podName,
				"-n", cr.namespace,
				"--", "sh", "-c", fmt.Sprintf("printenv %s", envVar),
			)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred(),
				fmt.Sprintf("failed to exec into pod %s to check %s", podName, envVar))
			g.Expect(strings.TrimSpace(string(output))).NotTo(BeEmpty(),
				fmt.Sprintf("%s should be non-empty in pod %s", envVar, podName))
		}
	}

	EventuallyWithOffset(1, validateEnvInPod, defaultTimeout, defaultPollPeriod).Should(Succeed())
}

// workloadKind returns the kind of workload the operator creates for the CR
func (cr crConfig) workloadKind() string {
	if cr.kind == "FalconNodeSensor" {
		return "daemonset"
	}
	return "deployment"
}

// workloadImages returns the init container and container images of the Deployment or DaemonSet
// the operator created for the CR
func (cr crConfig) workloadImages() ([]string, error) {
	cmd := exec.Command("kubectl", "get", cr.workloadKind(),
		"-n", cr.namespace,
		"-l", fmt.Sprintf("crowdstrike.com/component=%s", cr.componentName),
		"-o", "jsonpath={.items[*].spec.template.spec.initContainers[*].image} {.items[*].spec.template.spec.containers[*].image}",
	)
	output, err := utils.Run(cmd)
	return strings.Fields(string(output)), err
}

// validateWorkloadImage waits until every container in the CR's Deployment or DaemonSet uses an image matching imageMatcher
func (cr crConfig) validateWorkloadImage(imageMatcher types.GomegaMatcher) {
	By(fmt.Sprintf("validating the %s %s container images", cr.kind, cr.workloadKind()))
	validateImages := func(g Gomega) {
		images, err := cr.workloadImages()
		g.Expect(err).NotTo(HaveOccurred())
		fmt.Printf("%s images: %v\n", cr.workloadKind(), images)
		g.Expect(images).NotTo(BeEmpty())
		g.Expect(images).To(HaveEach(imageMatcher))
	}
	EventuallyWithOffset(1, validateImages, defaultTimeout, defaultPollPeriod).Should(Succeed())
}

// validateNotDeployed checks over duration that the operator neither creates the CR's Deployment
// or DaemonSet nor reports the Success condition
func (cr crConfig) validateNotDeployed(duration time.Duration) {
	By(fmt.Sprintf("validating that %s is not deployed for %v", cr.kind, duration))
	notDeployed := func(g Gomega) {
		images, err := cr.workloadImages()
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(images).To(BeEmpty())

		cmd := exec.Command("kubectl", "get", strings.ToLower(cr.kind),
			cr.metadataName, "-o", "jsonpath={.status.conditions[?(@.type==\"Success\")].status}",
			"-n", cr.namespace,
		)
		status, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(status)).NotTo(Equal("True"))
	}
	ConsistentlyWithOffset(1, notDeployed, duration, defaultPollPeriod).Should(Succeed())
}

// deleteCrInstance deletes the CR by name, if it exists, and waits until its install namespace no longer exists
func (cr crConfig) deleteCrInstance() {
	By(fmt.Sprintf("deleting %s %s", cr.kind, cr.metadataName))
	cmd := exec.Command("kubectl", "delete", strings.ToLower(cr.kind), cr.metadataName,
		"-n", cr.namespace, "--ignore-not-found=true", "--timeout=120s")
	_, err := utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By(fmt.Sprintf("waiting for %s namespace to be fully deleted", cr.namespace))
	namespaceDeleted := func() error {
		cmd := exec.Command("kubectl", "get", "namespace", cr.namespace, "--ignore-not-found=true", "-o", "name")
		output, err := utils.Run(cmd)
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(output)) != "" {
			return fmt.Errorf("namespace %s still exists", cr.namespace)
		}
		return nil
	}
	EventuallyWithOffset(1, namespaceDeleted, 5*time.Minute, defaultPollPeriod).Should(Succeed())
}
