package e2e

import (
	"encoding/json"
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
	namespace              = "falcon-operator-system"
	defaultTimeout         = 2 * time.Minute
	defaultPollPeriod      = 5 * time.Second
	metricsServiceName     = "falcon-operator-controller-manager-metrics-service"
	metricsRoleBindingName = "falcon-operator-metrics-binding"
	serviceAccountName     = "falcon-operator-controller-manager"
	metricsReaderRoleName  = "falcon-operator-metrics-reader"
)

var _ = Describe("falcon", Ordered, func() {
	BeforeAll(func() {
		// The namespace can be created when we run make install
		// However, in this test we want ensure that the solution
		// can run in a ns labeled as privileged. Therefore, we are
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
		cmd := exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
		By("removing metrics cluster role binding")
		cmd = exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName)
		_, _ = utils.Run(cmd)
	})

	Context("Falcon Operator", func() {
		var controllerPodName string
		It("should run successfully", func() {

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

			By("validating that manager Pod/container(s) are not restricted")
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
			EventuallyWithOffset(1, verifyControllerUp, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				fmt.Sprintf("--clusterrole=%s", metricsReaderRoleName),
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
			}
			EventuallyWithOffset(1, verifyMetricsEndpointReady, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("controller-runtime.metrics\tServing metrics server"),
					"Metrics server not yet started")
			}
			EventuallyWithOffset(1, verifyMetricsServerStarted, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": ["curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics"],
							"securityContext": {
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccount": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Succeeded"), "curl pod in wrong status")
			}
			// Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())
			EventuallyWithOffset(1, verifyCurlUp, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			metricsOutput := getMetricsOutput()
			Expect(metricsOutput).To(ContainSubstring(
				"controller_runtime_reconcile_total",
			))
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
			}, defaultTimeout, defaultPollPeriod).Should(Succeed())

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
			EventuallyWithOffset(1, getFalconNodeSensorPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())

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
			Eventually(getStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())
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
			}, defaultTimeout, defaultPollPeriod).Should(Succeed())

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
			EventuallyWithOffset(1, getFalconNodeSensorPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})
	})

	Context("Falcon Admission Controller", func() {
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
					"./config/samples/falcon_v1alpha1_falconadmission.yaml"),
					"client_id: PLEASE_FILL_IN", fmt.Sprintf("client_id: %s", falconClientID))
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				err = utils.ReplaceInFile(filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falconadmission.yaml"),
					"client_secret: PLEASE_FILL_IN", fmt.Sprintf("client_secret: %s", falconClientSecret))
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
			}

			By("creating an instance of the FalconAdmission Operand(CR)")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falconadmission.yaml"), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that pod(s) status.phase=Running")
			getFalconSidecarPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=admission_controller",
					"-o", "jsonpath={.items[*].status}", "-n", namespace,
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "\"phase\":\"Running\"") {
					return fmt.Errorf(" pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconSidecarPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that the status of the custom resource created is updated or not")
			getStatus := func() error {
				cmd := exec.Command("kubectl", "get", "falconadmission",
					"falcon-admission", "-A", "-o", "jsonpath={.status.conditions}",
					"-n", namespace,
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "Success") {
					return fmt.Errorf("status condition with type Success should be set")
				}
				return nil
			}
			Eventually(getStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})
	})

	Context("Falcon Admission Controller", func() {
		It("should cleanup successfully", func() {
			projectDir, _ := utils.GetProjectDir()

			By("deleting an instance of the FalconAdmission Operand(CR)")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "delete", "-f", filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falconadmission.yaml"), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that pod(s) status.phase!=Running")
			getFalconAdmissionPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=admission_controller", "--field-selector=status.phase=Running",
					"-o", "jsonpath={.items[*].status}", "-n", namespace,
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if len(status) > 0 {
					return fmt.Errorf("falcon-admission pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconAdmissionPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})
	})

	Context("Falcon Sidecar Sensor", func() {
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
					"./config/samples/falcon_v1alpha1_falconcontainer.yaml"),
					"client_id: PLEASE_FILL_IN", fmt.Sprintf("client_id: %s", falconClientID))
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				err = utils.ReplaceInFile(filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falconcontainer.yaml"),
					"client_secret: PLEASE_FILL_IN", fmt.Sprintf("client_secret: %s", falconClientSecret))
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
			}

			By("creating an instance of the FalconContainer Operand(CR)")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falconcontainer.yaml"), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that pod(s) status.phase=Running")
			getFalconNodeSensorPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=container_sensor",
					"-o", "jsonpath={.items[*].status}", "-n", namespace,
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "\"phase\":\"Running\"") {
					return fmt.Errorf(" pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconNodeSensorPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that the status of the custom resource created is updated or not")
			getStatus := func() error {
				cmd := exec.Command("kubectl", "get", "falconcontainer",
					"falcon-sidecar-sensor", "-A", "-o", "jsonpath={.status.conditions}",
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
			Eventually(getStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})
	})

	Context("Falcon Sidecar Sensor", func() {
		It("should cleanup successfully", func() {
			projectDir, _ := utils.GetProjectDir()

			By("deleting an instance of the FalconContainer Operand(CR)")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "delete", "-f", filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falconcontainer.yaml"), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that pod(s) status.phase!=Running")
			getFalconNodeSensorPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=container_sensor", "--field-selector=status.phase=Running",
					"-o", "jsonpath={.items[*].status}", "-n", namespace,
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if len(status) > 0 {
					return fmt.Errorf("falcon-sidecar-sensor pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconNodeSensorPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})
	})

	Context("Falcon Deployment Controller with Container Sensor", func() {
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
					"./config/samples/falcon_v1alpha1_falcondeployment-container-sensor.yaml"),
					"client_id: PLEASE_FILL_IN", fmt.Sprintf("client_id: %s", falconClientID))
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				err = utils.ReplaceInFile(filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falcondeployment-container-sensor.yaml"),
					"client_secret: PLEASE_FILL_IN", fmt.Sprintf("client_secret: %s", falconClientSecret))
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
			}

			By("creating an instance of the FalconDeployment Operand(CR) with Container Sensor")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falcondeployment-container-sensor.yaml"), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that FalconAdmission pod(s) status.phase=Running")
			getFalconAdmissionPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=admission_controller",
					"-o", "jsonpath={.items[*].status}", "-n", "falcon-kac",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "\"phase\":\"Running\"") {
					return fmt.Errorf(" pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconAdmissionPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that the status of the custom resource FalconAdmission created is updated or not")
			getFinalStatusAdmission := func() error {
				cmd := exec.Command("kubectl", "get", "falconadmission",
					"falcon-kac", "-A", "-o", "jsonpath={.status.conditions}",
					"-n", "falcon-kac",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "Success") {
					return fmt.Errorf("status condition with type Success should be set")
				}
				return nil
			}
			Eventually(getFinalStatusAdmission, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that FalconContainer pod(s) status.phase=Running")
			getFalconContainerSensorPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=container_sensor",
					"-o", "jsonpath={.items[*].status}", "-n", "falcon-system",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "\"phase\":\"Running\"") {
					return fmt.Errorf("falcon-container-sensor pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconContainerSensorPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that the status of the custom resource FalconContainer created is updated or not")
			getFinalStatusContainer := func() error {
				cmd := exec.Command("kubectl", "get", "falconcontainer",
					"falcon-container-sensor", "-A", "-o", "jsonpath={.status.conditions}",
					"-n", "falcon-system",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "Success") {
					return fmt.Errorf("status condition with type Success should be set")
				}
				return nil
			}
			Eventually(getFinalStatusContainer, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that FalconImageAnalyzer pod(s) status.phase=Running")
			getFalconImageAnalyzerPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=falcon-imageanalyzer",
					"-o", "jsonpath={.items[*].status}", "-n", "falcon-iar",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "\"phase\":\"Running\"") {
					return fmt.Errorf("falcon-imageanalyzer pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconImageAnalyzerPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that the status of the custom resource FalconImageAnalyzer created is updated or not")
			getFinalStatusIAR := func() error {
				cmd := exec.Command("kubectl", "get", "falconimageanalyzers",
					"falcon-image-analyzer", "-A", "-o", "jsonpath={.status.conditions}",
					"-n", "falcon-iar",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "Success") {
					return fmt.Errorf("status condition with type Success should be set")
				}
				return nil
			}
			Eventually(getFinalStatusIAR, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})
	})

	Context("Falcon Deployment Controller with Container Sensor", func() {
		It("should cleanup successfully", func() {
			projectDir, _ := utils.GetProjectDir()

			By("deleting an instance of the FalconDeployment Operand(CR)")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "delete", "-f", filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falcondeployment-container-sensor.yaml"), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that FalconAdmission pod(s) status.phase!=Running")
			getFalconAdmissionPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=admission_controller", "--field-selector=status.phase=Running",
					"-o", "jsonpath={.items[*].status}", "-n", "falcon-kac",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if len(status) > 0 {
					return fmt.Errorf("falcon-admission pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconAdmissionPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that pod(s) status.phase!=Running")
			getFalconContainerPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=container_sensor", "--field-selector=status.phase=Running",
					"-o", "jsonpath={.items[*].status}", "-n", "falcon-system",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if len(status) > 0 {
					return fmt.Errorf("falcon-container-sensor pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconContainerPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})
	})

	Context("Falcon Deployment Controller with Node Sensor", func() {
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
					"./config/samples/falcon_v1alpha1_falcondeployment-node-sensor.yaml"),
					"client_id: PLEASE_FILL_IN", fmt.Sprintf("client_id: %s", falconClientID))
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				err = utils.ReplaceInFile(filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falcondeployment-node-sensor.yaml"),
					"client_secret: PLEASE_FILL_IN", fmt.Sprintf("client_secret: %s", falconClientSecret))
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
			}

			By("creating an instance of the FalconDeployment Operand(CR) with Node Sensor")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falcondeployment-node-sensor.yaml"), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that FalconAdmission pod(s) status.phase=Running")
			getFalconAdmissionPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=admission_controller",
					"-o", "jsonpath={.items[*].status}", "-n", "falcon-kac",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "\"phase\":\"Running\"") {
					return fmt.Errorf(" pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconAdmissionPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that the status of the custom resource FalconAdmission created is updated or not")
			getFinalStatusAdmission := func() error {
				cmd := exec.Command("kubectl", "get", "falconadmission",
					"falcon-kac", "-A", "-o", "jsonpath={.status.conditions}",
					"-n", "falcon-kac",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "Success") {
					return fmt.Errorf("status condition with type Success should be set")
				}
				return nil
			}
			Eventually(getFinalStatusAdmission, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that FalconNodeSensor pod(s) status.phase=Running")
			getFalconNodeSensorPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=kernel_sensor",
					"-o", "jsonpath={.items[*].status}", "-n", "falcon-system",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "\"phase\":\"Running\"") {
					return fmt.Errorf("falcon-node-sensor pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconNodeSensorPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that the status of the custom resource FalconNodeSensor created is updated or not")
			getFinalStatusNode := func() error {
				cmd := exec.Command("kubectl", "get", "falconnodesensor",
					"falcon-node-sensor", "-A", "-o", "jsonpath={.status.conditions}",
					"-n", "falcon-system",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "Success") {
					return fmt.Errorf("status condition with type Success should be set")
				}
				return nil
			}
			Eventually(getFinalStatusNode, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that FalconImageAnalyzer pod(s) status.phase=Running")
			getFalconImageAnalyzerPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=falcon-imageanalyzer",
					"-o", "jsonpath={.items[*].status}", "-n", "falcon-iar",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "\"phase\":\"Running\"") {
					return fmt.Errorf("falcon-imageanalyzer pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconImageAnalyzerPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that the status of the custom resource FalconImageAnalyzer created is updated or not")
			getFinalStatusIAR := func() error {
				cmd := exec.Command("kubectl", "get", "falconimageanalyzers",
					"falcon-image-analyzer", "-A", "-o", "jsonpath={.status.conditions}",
					"-n", "falcon-iar",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "Success") {
					return fmt.Errorf("status condition with type Success should be set")
				}
				return nil
			}
			Eventually(getFinalStatusIAR, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})
	})

	Context("Falcon Deployment Controller with Node Sensor", func() {
		It("should cleanup successfully", func() {
			projectDir, _ := utils.GetProjectDir()

			By("deleting an instance of the FalconDeployment Operand(CR)")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "delete", "-f", filepath.Join(projectDir,
					"./config/samples/falcon_v1alpha1_falcondeployment-node-sensor.yaml"), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that FalconAdmission pod(s) status.phase!=Running")
			getFalconAdmissionPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=admission_controller", "--field-selector=status.phase=Running",
					"-o", "jsonpath={.items[*].status}", "-n", "falcon-kac",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if len(status) > 0 {
					return fmt.Errorf("falcon-admission pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconAdmissionPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that pod(s) status.phase!=Running")
			getFalconNodeSensorPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=kernel_sensor", "--field-selector=status.phase=Running",
					"-o", "jsonpath={.items[*].status}", "-n", "falcon-system",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if len(status) > 0 {
					return fmt.Errorf("falcon-node-sensor pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconNodeSensorPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validating that pod(s) status.phase!=Running")
			getFalconImageAnalyzerPodStatus := func() error {
				cmd := exec.Command("kubectl", "get",
					"pods", "-A", "-l", "crowdstrike.com/component=falcon-imageanalyzer", "--field-selector=status.phase=Running",
					"-o", "jsonpath={.items[*].status}", "-n", "falcon-iar",
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if len(status) > 0 {
					return fmt.Errorf("falcon-imageanalyzer pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getFalconImageAnalyzerPodStatus, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})
	})
})

func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() string {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	metricsOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return string(metricsOutput)
}

type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
