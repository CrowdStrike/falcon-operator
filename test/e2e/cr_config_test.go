package e2e

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/crowdstrike/falcon-operator/test/utils"
	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"
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
	iarDeploymentConfig = crConfig{
		kind:          "FalconImageAnalyzer",
		namespace:     "falcon-iar",
		metadataName:  "falcon-image-analyzer",
		componentName: "admission_controller",
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
		cmd := exec.Command("kubectl", "get", strings.ToLower(cr.kind),
			cr.metadataName, "-A", "-o", "jsonpath={.status.conditions}",
			"-n", cr.namespace,
		)
		status, err := utils.Run(cmd)
		fmt.Println(string(status))
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		if !strings.Contains(string(status), "Success") {
			return fmt.Errorf("status condition with type Success should be set")
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
	getNamespaceStatus := func() error {
		cmd := exec.Command("kubectl", "get", "namespace")
		status, err := utils.Run(cmd)
		ExpectWithOffset(3, err).NotTo(HaveOccurred())
		if strings.Contains(string(status), cr.namespace) {
			return fmt.Errorf("%s namespace still exists", cr.namespace)
		}
		return nil
	}
	EventuallyWithOffset(2, getNamespaceStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())
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
