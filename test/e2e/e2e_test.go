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
	namespace              = "falcon-operator-system"
	defaultTimeout         = 3 * time.Minute
	defaultPollPeriod      = 5 * time.Second
	metricsServiceName     = "falcon-operator-controller-manager-metrics-service"
	metricsRoleBindingName = "falcon-operator-metrics-binding"
	serviceAccountName     = "falcon-operator-controller-manager"
	metricsReaderRoleName  = "falcon-operator-metrics-reader"
	falconSecretNamespace  = "falcon-secrets"
	falconSecretName       = "falcon-secret"
	shouldBeRunning        = true
	shouldBeTerminated     = false
)

var _ = Describe("falcon", Ordered, func() {
	BeforeAll(func() {
		// The namespace can be created when we run make install
		// However, in this test we want to ensure that the solution
		// can run in a ns labeled as privileged. Therefore, we are
		// creating the namespace and labeling it.
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
		// Check if BUNDLE_IMG was used for installation and clean up accordingly
		bundleImg := os.Getenv("BUNDLE_IMG")
		if bundleImg != "" {
			By("cleaning up OLM bundle installation")

			// Get the proper operator-sdk executable path (following Makefile logic)
			operatorSDKPath, err := getOperatorSDKPath()
			if err == nil {
				cmd := exec.Command(operatorSDKPath, "cleanup", "falcon-operator", "--namespace", namespace)
				_, _ = utils.Run(cmd)

				// Uninstall OLM only if not on OpenShift (OpenShift has built-in OLM)
				if !isOpenShift() {
					By("uninstalling OLM")
					cmd = exec.Command(operatorSDKPath, "olm", "uninstall")
					_, _ = utils.Run(cmd)
				} else {
					By("detected OpenShift - skipping OLM uninstall (managed by OpenShift)")
				}
			} else {
				By("operator-sdk not found for cleanup, attempting manual cleanup")
			}
		} else {
			By("cleaning up traditional deployment")
			cmd := exec.Command("make", "undeploy")
			_, _ = utils.Run(cmd)
		}

		By("removing manager namespace")
		cmd := exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
		cmd = exec.Command("kubectl", "delete", "ns", falconSecretNamespace)
		_, _ = utils.Run(cmd)
		By("removing metrics cluster role binding")
		cmd = exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName)
		_, _ = utils.Run(cmd)

		// Only run make install for traditional deployment cleanup
		if bundleImg == "" {
			cmd = exec.Command("make", "install")
			_, _ = utils.Run(cmd)
		}
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

			var outputMake []byte
			var cmd *exec.Cmd

			// Conditional installation: Use OLM bundle if BUNDLE_IMG is set, otherwise use traditional deployment
			// This allows testing both installation methods:
			// - OLM bundle: Set BUNDLE_IMG=<bundle-image-url> to test operator-sdk run bundle installation
			// - Traditional: Leave BUNDLE_IMG unset to use make deploy with custom operator image
			bundleImg := os.Getenv("BUNDLE_IMG")
			if bundleImg != "" {
				By("installing using OLM bundle: " + bundleImg)

				// Get the proper operator-sdk executable path (following Makefile logic)
				operatorSDKPath, err := getOperatorSDKPath()
				if err != nil {
					ExpectWithOffset(1, err).NotTo(HaveOccurred(), "operator-sdk is required when BUNDLE_IMG is set")
				}

				// Check if operator-sdk is available and working
				cmd = exec.Command(operatorSDKPath, "version")
				_, err = utils.Run(cmd)
				if err != nil {
					ExpectWithOffset(1, err).NotTo(HaveOccurred(), "operator-sdk is required when BUNDLE_IMG is set")
				}

				// Install OLM if not already present (skip on OpenShift as it has OLM built-in)
				if !isOpenShift() {
					By("installing OLM")
					cmd = exec.Command(operatorSDKPath, "olm", "install")
					_, err = utils.Run(cmd)
					if err != nil {
						By("OLM may already be installed, continuing...")
					}
				} else {
					By("detected OpenShift - skipping OLM installation (already built-in)")
				}

				// Run bundle installation
				By("deploying operator via OLM bundle")
				cmd = exec.Command(operatorSDKPath, "run", "bundle", bundleImg, "--namespace", namespace)
				outputMake, err = utils.Run(cmd)
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
			} else {
				By("installing using traditional deployment method")

				cmd = exec.Command("kind", "get", "clusters")
				_, err = utils.Run(cmd)
				if err == nil {
					By("building the manager (Operator) image")
					cmd = exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", operatorImage))
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
				outputMake, err = utils.Run(cmd)
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
			}

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
		manifest := "./config/samples/falcon_v1alpha1_falconnodesensor.yaml"
		It("should deploy successfully", func() {
			updateManifestApiCreds(manifest)
			nodeConfig.manageCrdInstance(crApply, manifest)
			nodeConfig.validateCrStatus()
		})
		It("should cleanup successfully", func() {
			nodeConfig.manageCrdInstance(crDelete, manifest)
			nodeConfig.validateRunningStatus(shouldBeTerminated)
			nodeConfig.waitForNamespaceDeletion()
		})
	})

	Context("Falcon Node Sensor - GKE Autopilot", func() {
		manifest := "./config/samples/falcon_v1alpha1_falconnodesensor-gke-autopilot.yaml"
		It("should deploy successfully", func() {
			updateManifestApiCreds(manifest)
			nodeConfig.manageCrdInstance(crApply, manifest)
			nodeConfig.validateCrStatus()
			nodeConfig.validateInitContainerReadOnlyRootFilesystem()
		})
		It("should cleanup successfully", func() {
			nodeConfig.manageCrdInstance(crDelete, manifest)
			nodeConfig.validateRunningStatus(shouldBeTerminated)
			nodeConfig.waitForNamespaceDeletion()
		})
	})

	Context("Falcon Admission Controller", func() {
		manifest := "./config/samples/falcon_v1alpha1_falconadmission.yaml"
		It("should deploy successfully", func() {
			updateManifestApiCreds(manifest)
			kacConfig.manageCrdInstance(crApply, manifest)
			kacConfig.validateRunningStatus(shouldBeRunning)
			kacConfig.validateCrStatus()
		})
	})

	Context("Falcon Admission Controller", func() {
		It("should manage falcon-kac-meta configMap changes successfully", func() {
			manifest := "./config/samples/falcon_v1alpha1_falconadmission_custom_clustername.yaml"
			updateManifestApiCreds(manifest)

			By("update with a clustom clusterName")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir, manifest), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("validate the cluster name in the falcon-kac-meta configMap has updated")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "get", "configmap", "falcon-kac-meta",
					"-n", kacConfig.namespace, "-o", "jsonpath='{.data.ClusterName}'")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "test-cluster") {
					return fmt.Errorf("falcon-admission pod configMap not updated: %s", output)
				}
				return nil
			}, defaultTimeout, defaultPollPeriod).Should(Succeed())

			kacConfig.validateRunningStatus(shouldBeRunning)
		})
	})

	Context("Falcon Admission Controller", func() {
		manifest := "./config/samples/falcon_v1alpha1_falconadmission.yaml"
		It("should cleanup successfully", func() {
			kacConfig.manageCrdInstance(crDelete, manifest)
			kacConfig.validateRunningStatus(shouldBeTerminated)
			kacConfig.waitForNamespaceDeletion()
		})
	})

	Context("Falcon Sidecar Sensor", func() {
		manifest := "./config/samples/falcon_v1alpha1_falconcontainer.yaml"
		It("should deploy successfully", func() {
			updateManifestApiCreds(manifest)
			sidecarConfig.manageCrdInstance(crApply, manifest)
			sidecarConfig.validateRunningStatus(shouldBeRunning)
			sidecarConfig.validateCrStatus()
		})
		It("should cleanup successfully", func() {
			sidecarConfig.manageCrdInstance(crDelete, manifest)
			sidecarConfig.validateRunningStatus(shouldBeTerminated)
			sidecarConfig.waitForNamespaceDeletion()
		})
	})

	Context("Falcon Sidecar Sensor with Falcon Secret", func() {
		manifest := "./config/samples/falcon_v1alpha1_falconcontainer-with-falcon-secret.yaml"
		It("should deploy successfully", func() {
			addFalconSecretToManifest(manifest)
			sidecarConfig.manageCrdInstance(crApply, manifest)
			sidecarConfig.validateRunningStatus(shouldBeRunning)
			sidecarConfig.validateCrStatus()
		})
		It("should cleanup successfully", func() {
			sidecarConfig.manageCrdInstance(crDelete, manifest)
			sidecarConfig.validateRunningStatus(shouldBeTerminated)
			secretConfig.deleteNamespace()
			sidecarConfig.waitForNamespaceDeletion()
			secretConfig.waitForNamespaceDeletion()
		})
	})

	Context("Falcon Deployment Controller with Container Sensor", func() {
		manifest := "./config/samples/falcon_v1alpha1_falcondeployment-container-sensor.yaml"
		It("should deploy successfully", func() {
			updateManifestApiCreds(manifest)
			falconDeploymentConfig.manageCrdInstance(crApply, manifest)
			kacConfig.validateRunningStatus(shouldBeRunning)
			kacConfig.validateCrStatus()
			sidecarConfig.validateRunningStatus(shouldBeRunning)
			sidecarConfig.validateCrStatus()
			iarConfig.validateRunningStatus(shouldBeRunning)
			iarConfig.validateCrStatus()
		})
		It("should cleanup successfully", func() {
			falconDeploymentConfig.manageCrdInstance(crDelete, manifest)
			kacConfig.validateRunningStatus(shouldBeTerminated)
			sidecarConfig.validateRunningStatus(shouldBeTerminated)
			iarConfig.validateRunningStatus(shouldBeTerminated)
			sidecarConfig.waitForNamespaceDeletion()
			kacConfig.waitForNamespaceDeletion()
			iarConfig.waitForNamespaceDeletion()
		})
	})

	Context("Falcon Deployment Controller with Node Sensor", func() {
		manifest := "./config/samples/falcon_v1alpha1_falcondeployment-node-sensor.yaml"
		It("should deploy successfully", func() {
			updateManifestApiCreds(manifest)
			falconDeploymentConfig.manageCrdInstance(crApply, manifest)
			kacConfig.validateRunningStatus(shouldBeRunning)
			kacConfig.validateCrStatus()
			nodeConfig.validateRunningStatus(shouldBeRunning)
			nodeConfig.validateCrStatus()
			iarConfig.validateRunningStatus(shouldBeRunning)
			iarConfig.validateCrStatus()
		})
		It("should cleanup successfully", func() {
			falconDeploymentConfig.manageCrdInstance(crDelete, manifest)
			kacConfig.validateRunningStatus(shouldBeTerminated)
			nodeConfig.validateRunningStatus(shouldBeTerminated)
			iarConfig.validateRunningStatus(shouldBeTerminated)
			nodeConfig.waitForNamespaceDeletion()
			kacConfig.waitForNamespaceDeletion()
			iarConfig.waitForNamespaceDeletion()
		})
	})

	Context("Falcon Deployment Controller with Node Sensor and Falcon Secret", func() {
		manifest := "./config/samples/falcon_v1alpha1_falcondeployment-node-sensor-with-falcon-secret.yaml"
		It("should deploy successfully", func() {
			addFalconSecretToManifest(manifest)
			falconDeploymentConfig.manageCrdInstance(crApply, manifest)
			kacConfig.validateRunningStatus(shouldBeRunning)
			kacConfig.validateCrStatus()
			nodeConfig.validateRunningStatus(shouldBeRunning)
			nodeConfig.validateCrStatus()
			iarConfig.validateRunningStatus(shouldBeRunning)
			iarConfig.validateCrStatus()
		})
		It("should cleanup successfully", func() {
			falconDeploymentConfig.manageCrdInstance(crDelete, manifest)
			kacConfig.validateRunningStatus(shouldBeTerminated)
			nodeConfig.validateRunningStatus(shouldBeTerminated)
			iarConfig.validateRunningStatus(shouldBeTerminated)
			secretConfig.deleteNamespace()
			nodeConfig.waitForNamespaceDeletion()
			kacConfig.waitForNamespaceDeletion()
			iarConfig.waitForNamespaceDeletion()
			secretConfig.waitForNamespaceDeletion()
		})
	})
})
