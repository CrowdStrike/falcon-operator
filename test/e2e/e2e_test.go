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

// constant parts of the file
const (
	namespace = "falcon-operator-system"
)

var _ = Describe("falcon", Ordered, func() {
	BeforeAll(func() {
		// The namespace can be created when we run make install
		// However, in this test we want ensure that the solution
		// can run in a ns labeled as restricted. Therefore, we are
		// creating the namespace an lebeling it.
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, _ = utils.Run(cmd)

		By("labeling enforce the namespace where the Operator and Operand(s) will run")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/audit=privileged",
			"pod-security.kubernetes.io/enforce-version=latest",
			"pod-security.kubernetes.io/enforce=privileged")
		_, err := utils.Run(cmd)
		Expect(err).To(Not(HaveOccurred()))
	})

	AfterAll(func() {
		By("removing manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	Context("Falcon Operator", func() {
		It("should run successfully", func() {
			var controllerPodName string
			var err error

			// operatorImage stores the name of the image used in the example
			var operatorImage = "example.com/falcon-operator:v0.0.1"
			if image, ok := os.LookupEnv("OPERATOR_IMAGE"); ok {
				operatorImage = image
			}

			cmd := exec.Command("kind", "get", "clusters")
			_, err = utils.Run(cmd)
			if err == nil {
				By("building the manager (Operator) image")
				cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", operatorImage))
				_, err = utils.Run(cmd)
				ExpectWithOffset(1, err).NotTo(HaveOccurred())

				By("loading the the manager(Operator) image on Kind")
				err = utils.LoadImageToKindClusterWithName(operatorImage)
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
			}

			By("installing CRDs")
			cmd = exec.Command("make", "install")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("deploying the controller-manager")
			cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", operatorImage))
			outputMake, err := utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			fmt.Println(outputMake)
			By("validating that manager Pod/container(s) are restricted")
			ExpectWithOffset(1, outputMake).NotTo(ContainSubstring("Warning: would violate PodSecurity"))

			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func() error {
				// Get pod name
				cmd = exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)
				podOutput, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(string(podOutput))
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

				// Validate pod status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				status, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if string(status) != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())

		})
	})

	Context("Falcon Node Sensor", func() {
		It("should deploy successfully", func() {
			projectDir, _ := utils.GetProjectDir()

			var falconClientID = ""
			var falconClientSecret = ""
			if clientID, ok := os.LookupEnv("FALCON_CLIENT_ID"); ok {
				falconClientID = clientID
			}

			if clientSecret, ok := os.LookupEnv("FALCON_CLIENT_SECRET"); ok {
				falconClientSecret = clientSecret
			}

			if falconClientID != "" && falconClientSecret != "" {
				err := utils.ReplaceInFile(filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falconnodesensor.yaml"),
					"client_id: PLEASE_FILL_IN", fmt.Sprintf("client_id: %s", falconClientID))
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				err = utils.ReplaceInFile(filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falconnodesensor.yaml"),
					"client_secret: PLEASE_FILL_IN", fmt.Sprintf("client_secret: %s", falconClientSecret))
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
			}

			By("creating an instance of the FalconNodeSensor Operand(CR)")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falconnodesensor.yaml"), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("validating that pod(s) status.phase=Running")
			getFalconNodeSensorPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=kernel_sensor",
					"-o", "jsonpath={.items[*].status}", "-n", namespace,
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "\"phase\":\"Running\"") {
					return fmt.Errorf("falcon-node-sensor pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconNodeSensorPodStatus, time.Minute, time.Second).Should(Succeed())

			By("validating that the status of the custom resource created is updated or not")
			getStatus := func() error {
				cmd := exec.Command("kubectl", "get", "falconnodesensor",
					"falcon-node-sensor", "-A", "-o", "jsonpath={.status.conditions}",
					"-n", namespace,
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "Success") {
					return fmt.Errorf("status condition with type Success should be set")
				}
				return nil
			}
			Eventually(getStatus, time.Minute, time.Second).Should(Succeed())
		})
	})

	Context("Falcon Node Sensor", func() {
		It("should cleanup successfully", func() {
			projectDir, _ := utils.GetProjectDir()

			By("deleting an instance of the FalconNodeSensor Operand(CR)")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "delete", "-f", filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falconnodesensor.yaml"), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("validating that pod(s) status.phase!=Running")
			getFalconNodeSensorPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=kernel_sensor", "--field-selector=status.phase=Running",
					"-o", "jsonpath={.items[*].status}", "-n", namespace,
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if len(status) > 0 {
					return fmt.Errorf("falcon-node-sensor pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconNodeSensorPodStatus, time.Minute, time.Second).Should(Succeed())
		})
	})
})
