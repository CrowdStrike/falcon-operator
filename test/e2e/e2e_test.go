package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/test/utils"
	corev1 "k8s.io/api/core/v1"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"
)

// Environment Variables:
//   USE_EXISTING_OPERATOR      - Set to "true" to run tests against an existing operator installation
//                                without installing OLM, building images, installing CRDs,
//                                or deploying the operator. When set, the tests will use
//                                the operator deployment already present in the cluster.
//   OPERATOR_NAMESPACE         - Override the namespace where the operator is deployed.
//                                Defaults to "falcon-operator" when USE_EXISTING_OPERATOR=true,
//                                otherwise defaults to "falcon-operator-system".
//   BUNDLE_IMG                 - (Optional) Use OLM bundle installation method instead of
//                                traditional make deploy. Only used when USE_EXISTING_OPERATOR
//                                is not set.
//   OPERATOR_IMAGE             - (Optional) Custom operator image to use. Only used when
//                                USE_EXISTING_OPERATOR is not set and BUNDLE_IMG is not set.
//   SKIP_TESTS                 - (Optional) Comma-delimited list of resource kinds to skip.
//                                Examples: "FalconNodeSensor", "FalconAdmission,FalconContainer"
//   RECONCILE_LOOP_CHECK       - (Optional) Set to "true" to enable reconcile loop validation.
//
// Example usage with existing operator installation:
//   USE_EXISTING_OPERATOR=true OPERATOR_NAMESPACE=falcon-operator go test ./test/e2e/...
//
// Example usage skipping specific tests:
//   SKIP_TESTS="FalconNodeSensor,FalconAdmission" go test ./test/e2e/...
//
// Example usage enabling reconcile loop checks:
//   RECONCILE_LOOP_CHECK=true go test ./test/e2e/...

// constant parts of the file
const (
	defaultTimeout                  = 3 * time.Minute
	defaultPollPeriod               = 5 * time.Second
	reconcileLoopValidationDuration = 30 * time.Second
	metricsServiceName              = "falcon-operator-controller-manager-metrics-service"
	metricsRoleBindingName          = "falcon-operator-metrics-binding"
	serviceAccountName              = "falcon-operator-controller-manager"
	metricsReaderRoleName           = "falcon-operator-metrics-reader"
	falconSecretNamespace           = "falcon-secrets"
	falconSecretName                = "falcon-secret"
	shouldBeRunning                 = true
	shouldBeTerminated              = false
)

var (
	namespace          string
	skipMetricsTest    bool
	controllerPodName  string
	reconcileLoopCheck bool
)

var _ = Describe("falcon", Ordered, func() {
	BeforeAll(func() {
		skipMetricsTest = false

		reconcileLoopCheck = os.Getenv("RECONCILE_LOOP_CHECK") == "true"
		if reconcileLoopCheck {
			By("RECONCILE_LOOP_CHECK: reconcile loop validation will be performed")
		} else {
			By("RECONCILE_LOOP_CHECK not set: skipping reconcile loop validation for faster tests")
		}

		useExistingOperator := os.Getenv("USE_EXISTING_OPERATOR") == "true"

		// Set namespace based on environment or default
		if ns := os.Getenv("OPERATOR_NAMESPACE"); ns != "" {
			namespace = ns
		} else if useExistingOperator {
			namespace = "falcon-operator"
		} else {
			namespace = "falcon-operator-system"
		}

		if useExistingOperator {
			By("using existing operator installation in namespace: " + namespace)
			// Skip namespace creation for existing operator installations

			// Check if metrics service exists in the cluster
			cmd := exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err := utils.Run(cmd)
			if err != nil {
				// Metrics service doesn't exist, skip metrics test
				skipMetricsTest = true
				By("metrics service not found in existing operator installation - skipping metrics test")
			}

			return
		}

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

	AfterEach(func() {
		if !reconcileLoopCheck {
			return
		}

		report := CurrentSpecReport()

		// Only run for "deploy successfully" tests, not "cleanup" tests
		if !strings.Contains(report.LeafNodeText, "should deploy successfully") {
			return
		}

		labels := report.Labels()

		// Handle FalconDeployment specially - it creates multiple resources
		if slices.Contains(labels, "FalconDeployment") {
			var wg sync.WaitGroup
			// Check all possible resources that FalconDeployment might create
			kinds := []string{falconDeploymentConfig.kind, kacConfig.kind, sidecarConfig.kind, nodeConfig.kind, iarConfig.kind}

			for _, kind := range kinds {
				wg.Add(1)
				go func(k string) {
					defer wg.Done()
					defer GinkgoRecover()
					validateNoReconcileLoop(controllerPodName, namespace, k, reconcileLoopValidationDuration)
				}(kind)
			}
			wg.Wait()
			return
		}

		// For single resource tests
		var kind string
		if slices.Contains(labels, "FalconNodeSensor") {
			kind = nodeConfig.kind
		} else if slices.Contains(labels, "FalconAdmission") {
			kind = kacConfig.kind
		} else if slices.Contains(labels, "FalconContainer") {
			kind = sidecarConfig.kind
		}

		if kind != "" {
			validateNoReconcileLoop(controllerPodName, namespace, kind, reconcileLoopValidationDuration)
		}
	})

	AfterAll(func() {
		useExistingOperator := os.Getenv("USE_EXISTING_OPERATOR") == "true"

		// Clean up CRD instances for all cluster types (existing operator or not)
		By("cleaning up CRD instances")

		// Delete FalconDeployment instances (which may include multiple sensors)
		By("deleting FalconDeployment instances")
		cmd := exec.Command("kubectl", "delete", "falcondeployment", "--all", "-A", "--timeout=60s", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		// Delete individual CRD instances
		By("deleting FalconNodeSensor instances")
		cmd = exec.Command("kubectl", "delete", "falconnodesensor", "--all", "-A", "--timeout=60s", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		By("deleting FalconAdmission instances")
		cmd = exec.Command("kubectl", "delete", "falconadmission", "--all", "-A", "--timeout=60s", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		By("deleting FalconContainer instances")
		cmd = exec.Command("kubectl", "delete", "falconcontainer", "--all", "-A", "--timeout=60s", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		By("deleting FalconImageAnalyzer instances")
		cmd = exec.Command("kubectl", "delete", "falconimageanalyzer", "--all", "-A", "--timeout=60s", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		// Clean up test-created namespaces
		By("cleaning up test namespaces")
		testNamespaces := []string{
			nodeConfig.namespace,  // falcon-system
			kacConfig.namespace,   // falcon-kac
			iarConfig.namespace,   // falcon-iar
			falconSecretNamespace, // falcon-secrets
		}

		for _, ns := range testNamespaces {
			By("deleting namespace: " + ns)
			cmd = exec.Command("kubectl", "delete", "ns", ns, "--timeout=300s", "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		}

		if useExistingOperator {
			By("skipping operator cleanup for existing operator installation")
			return
		}

		// For non-existing operator installations, also clean up the operator
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
		cmd = exec.Command("kubectl", "delete", "ns", namespace, "--timeout=300s")
		_, _ = utils.Run(cmd)

		By("removing metrics cluster role binding")
		cmd = exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		// Only run make install for traditional deployment cleanup
		if bundleImg == "" {
			cmd = exec.Command("make", "uninstall")
			_, _ = utils.Run(cmd)
		}
	})

	Context("Falcon Operator", Label("FalconNodeSensor", "FalconAdmission", "FalconContainer", "FalconDeployment"), func() {
		It("should run successfully", func() {

			var err error
			useExistingOperator := os.Getenv("USE_EXISTING_OPERATOR") == "true"

			// Skip installation if using existing operator
			if useExistingOperator {
				By("skipping operator installation - using existing operator deployment")
			} else {
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
			}

			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func() error {
				// Get pod name
				cmd := exec.Command("kubectl", "get",
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
			if skipMetricsTest {
				By("skipping metrics test - metrics service not available in non-OLM installation by default")
				Skip("Metrics test skipped due to missing metrics service")
			}

			// Skip ClusterRoleBinding creation on OpenShift as it may already exist or have different permissions
			if !isOpenShift() {
				By("creating a ClusterRoleBinding for the service account to allow access to metrics")
				cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
					fmt.Sprintf("--clusterrole=%s", metricsReaderRoleName),
					fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
				)
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

				// Ensure cleanup of ClusterRoleBinding even if test fails
				DeferCleanup(func() {
					By("cleaning up test ClusterRoleBinding")
					cmd := exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName, "--ignore-not-found=true")
					_, _ = utils.Run(cmd)
				})
			} else {
				By("skipping ClusterRoleBinding creation on OpenShift - using existing permissions")
			}

			By("validating that the metrics service is available")
			cmd := exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err := utils.Run(cmd)
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

			// Skip curl-metrics pod on OpenShift due to security restrictions
			if !isOpenShift() {
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

				// Ensure cleanup of curl-metrics pod
				DeferCleanup(func() {
					By("cleaning up curl-metrics pod")
					cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace, "--ignore-not-found=true")
					_, _ = utils.Run(cmd)
				})

				By("waiting for the curl-metrics pod to complete.")
				verifyCurlUp := func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
						"-o", "jsonpath={.status.phase}",
						"-n", namespace)
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).To(ContainSubstring("Succeeded"), "curl pod in wrong status")
				}
				EventuallyWithOffset(1, verifyCurlUp, defaultTimeout, defaultPollPeriod).Should(Succeed())

				By("getting the metrics by checking curl-metrics logs")
				metricsOutput := getMetricsOutput()
				Expect(metricsOutput).To(ContainSubstring(
					"controller_runtime_reconcile_total",
				))
			} else {
				By("skipping curl-metrics pod on OpenShift - metrics endpoint verified via logs")
			}
		})
	})

	Context("Falcon Node Sensor", Label("FalconNodeSensor"), func() {
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

	Context("Falcon Node Sensor - GKE Autopilot", Label("FalconNodeSensor"), func() {
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

	Context("Falcon Admission Controller", Label("FalconAdmission"), func() {
		manifest := "./config/samples/falcon_v1alpha1_falconadmission.yaml"
		It("should deploy successfully", func() {
			updateManifestApiCreds(manifest)
			kacConfig.manageCrdInstance(crApply, manifest)
			kacConfig.validateRunningStatus(shouldBeRunning)
			kacConfig.validateCrStatus()
		})
	})

	Context("Falcon Admission Controller", Label("FalconAdmission"), func() {
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

	Context("Falcon Admission Controller", Label("FalconAdmission"), func() {
		manifest := "./config/samples/falcon_v1alpha1_falconadmission.yaml"
		It("should cleanup successfully", func() {
			kacConfig.manageCrdInstance(crDelete, manifest)
			kacConfig.validateRunningStatus(shouldBeTerminated)
			kacConfig.waitForNamespaceDeletion()
		})
	})

	Context("Falcon Sidecar Sensor", Label("FalconContainer"), func() {
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

	Context("Falcon Sidecar Sensor with Falcon Secret", Label("FalconContainer"), func() {
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

	Context("Falcon Sidecar Sensor with AITap", Label("FalconContainer"), func() {
		manifest := "./config/samples/falcon_v1alpha1_falconcontainer-with-aitap.yaml"
		It("should deploy successfully", func() {
			updateManifestApiCreds(manifest)
			updateManifestWithAITapToken(manifest)
			updateManifestWithAITapBaseURL(manifest)
			sidecarConfig.manageCrdInstance(crApply, manifest)
			sidecarConfig.validateRunningStatus(shouldBeRunning)
			sidecarConfig.validateCrStatus()
			validateAITapSecrets()
		})
		It("should cleanup successfully", func() {
			sidecarConfig.manageCrdInstance(crDelete, manifest)
			sidecarConfig.validateRunningStatus(shouldBeTerminated)
			sidecarConfig.waitForNamespaceDeletion()
		})
	})

	Context("Falcon Deployment Controller with Container Sensor", Label("FalconDeployment"), func() {
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

	Context("Falcon Deployment Controller with Node Sensor", Label("FalconDeployment"), func() {
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

	Context("Falcon Deployment Controller with Node Sensor and Falcon Secret", Label("FalconDeployment"), func() {
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

	Context("Falcon Image Analyzer Tolerations", Label("FalconImageAnalyzer"), func() {
		manifest := "./config/samples/falcon_v1alpha1_falconimageanalyzer.yaml"
		It("should deploy with tolerations successfully", func() {
			By("loading and modifying the FalconImageAnalyzer manifest")
			var iar falconv1alpha1.FalconImageAnalyzer
			err := loadManifest(manifest, &iar)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			// Add tolerations at the spec level
			iar.Spec.Tolerations = []corev1.Toleration{
				{
					Key:               "node.kubernetes.io/not-ready",
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: func(i int64) *int64 { return &i }(300),
				},
			}

			By("applying the modified manifest")
			err = applyManifest(&iar, iarConfig.namespace)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			iarConfig.validateRunningStatus(shouldBeRunning)
			iarConfig.validateCrStatus()

			By("validating the deployment has the expected tolerations")
			validateTolerationsInDeployment := func() error {
				cmd := exec.Command("kubectl", "get", "deployment", "falcon-image-analyzer",
					"-n", iarConfig.namespace,
					"-o", "jsonpath={.spec.template.spec.tolerations[?(@.key=='node.kubernetes.io/not-ready')].effect}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "NoExecute") {
					return fmt.Errorf("expected toleration not found in deployment: %s", output)
				}
				return nil
			}
			EventuallyWithOffset(1, validateTolerationsInDeployment, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})

		It("should replace toleration when Key+Effect match but Value/Operator differ", func() {
			By("loading and modifying the FalconImageAnalyzer manifest with initial toleration")
			var iar falconv1alpha1.FalconImageAnalyzer
			err := loadManifest(manifest, &iar)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			// Set initial toleration with Exists operator
			iar.Spec.Tolerations = []corev1.Toleration{
				{
					Key:      "app",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
			}

			By("applying the manifest with initial toleration")
			err = applyManifest(&iar, iarConfig.namespace)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating initial toleration with Exists operator")
			validateInitialToleration := func() error {
				cmd := exec.Command("kubectl", "get", "deployment", "falcon-image-analyzer",
					"-n", iarConfig.namespace,
					"-o", "jsonpath={.spec.template.spec.tolerations[?(@.key=='app')].operator}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "Exists") {
					return fmt.Errorf("expected Exists operator not found: %s", output)
				}
				return nil
			}
			EventuallyWithOffset(1, validateInitialToleration, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("updating toleration with same Key+Effect but different Operator and Value")
			err = loadManifest(manifest, &iar)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			// Update to Equal operator with value for same Key+Effect
			iar.Spec.Tolerations = []corev1.Toleration{
				{
					Key:      "app",
					Operator: corev1.TolerationOpEqual,
					Value:    "falcon",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			}

			By("applying the updated manifest")
			err = applyManifest(&iar, iarConfig.namespace)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating toleration was replaced with new Value and Operator")
			validateReplacedToleration := func() error {
				// Check for new operator
				cmd := exec.Command("kubectl", "get", "deployment", "falcon-image-analyzer",
					"-n", iarConfig.namespace,
					"-o", "jsonpath={.spec.template.spec.tolerations[?(@.key=='app')].operator}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "Equal") {
					return fmt.Errorf("expected Equal operator not found: %s", output)
				}

				// Check for new value
				cmd = exec.Command("kubectl", "get", "deployment", "falcon-image-analyzer",
					"-n", iarConfig.namespace,
					"-o", "jsonpath={.spec.template.spec.tolerations[?(@.key=='app')].value}")
				output, err = utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "falcon") {
					return fmt.Errorf("expected value 'falcon' not found: %s", output)
				}

				// Verify only one toleration with key 'app' exists
				cmd = exec.Command("kubectl", "get", "deployment", "falcon-image-analyzer",
					"-n", iarConfig.namespace,
					"-o", "jsonpath={.spec.template.spec.tolerations[?(@.key=='app')]}")
				output, err = utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				// Should only have one match (not duplicate)
				matches := strings.Count(string(output), `"key":"app"`)
				if matches > 1 {
					return fmt.Errorf("found %d tolerations with key 'app', expected 1: %s", matches, output)
				}
				return nil
			}
			EventuallyWithOffset(1, validateReplacedToleration, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})

		It("should allow multiple tolerations with same Key but different Effect", func() {
			By("loading and modifying the FalconImageAnalyzer manifest with multiple tolerations")
			var iar falconv1alpha1.FalconImageAnalyzer
			err := loadManifest(manifest, &iar)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			// Set multiple tolerations with same Key but different Effects
			iar.Spec.Tolerations = []corev1.Toleration{
				{
					Key:      "node-role",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
				{
					Key:      "node-role",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoExecute,
				},
			}

			By("applying the manifest with multiple tolerations")
			err = applyManifest(&iar, iarConfig.namespace)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating both tolerations with same Key but different Effect exist")
			validateBothEffects := func() error {
				// Get all tolerations with key 'node-role'
				cmd := exec.Command("kubectl", "get", "deployment", "falcon-image-analyzer",
					"-n", iarConfig.namespace,
					"-o", "jsonpath={.spec.template.spec.tolerations[?(@.key=='node-role')]}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())

				// Verify both effects are present
				outputStr := string(output)
				if !strings.Contains(outputStr, "NoSchedule") {
					return fmt.Errorf("NoSchedule effect not found for key 'node-role': %s", outputStr)
				}
				if !strings.Contains(outputStr, "NoExecute") {
					return fmt.Errorf("NoExecute effect not found for key 'node-role': %s", outputStr)
				}

				// Verify we have exactly 2 tolerations with this key (one for each effect)
				effectCount := strings.Count(outputStr, `"effect":`)
				if effectCount != 2 {
					return fmt.Errorf("expected 2 tolerations with key 'node-role', found %d: %s", effectCount, outputStr)
				}
				return nil
			}
			EventuallyWithOffset(1, validateBothEffects, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})

		It("should cleanup successfully", func() {
			iarConfig.manageCrdInstance(crDelete, manifest)
			iarConfig.validateRunningStatus(shouldBeTerminated)
			iarConfig.waitForNamespaceDeletion()
		})
	})

	Context("Falcon Admission Controller Tolerations", Label("FalconAdmission"), func() {
		manifest := "./config/samples/falcon_v1alpha1_falconadmission.yaml"
		It("should deploy successfully", func() {
			By("loading and modifying the FalconAdmission manifest")
			var admission falconv1alpha1.FalconAdmission
			err := loadManifest(manifest, &admission)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			// Add tolerations to AdmissionConfig
			admission.Spec.AdmissionConfig.Tolerations = []corev1.Toleration{
				{
					Key:      "node.kubernetes.io/memory-pressure",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
			}

			By("applying the modified manifest")
			err = applyManifest(&admission, kacConfig.namespace)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			kacConfig.validateRunningStatus(shouldBeRunning)
			kacConfig.validateCrStatus()

			By("validating the deployment has the expected tolerations")
			validateTolerationsInDeployment := func() error {
				cmd := exec.Command("kubectl", "get", "deployment", "falcon-kac",
					"-n", kacConfig.namespace,
					"-o", "jsonpath={.spec.template.spec.tolerations[?(@.key=='node.kubernetes.io/memory-pressure')].effect}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "NoSchedule") {
					return fmt.Errorf("expected toleration not found in deployment: %s", output)
				}
				return nil
			}
			EventuallyWithOffset(1, validateTolerationsInDeployment, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})

		It("should preserve system-added tolerations", func() {
			By("getting current tolerations")
			cmd := exec.Command("kubectl", "get", "deployment", "falcon-kac",
				"-n", kacConfig.namespace,
				"-o", "jsonpath={.spec.template.spec.tolerations}")
			currentTolerations, err := utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("loading and modifying the FalconAdmission manifest with additional tolerations")
			var admission falconv1alpha1.FalconAdmission
			err = loadManifest(manifest, &admission)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			// Add multiple tolerations to AdmissionConfig
			admission.Spec.AdmissionConfig.Tolerations = []corev1.Toleration{
				{
					Key:      "node.kubernetes.io/memory-pressure",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
				{
					Key:      "custom-taint",
					Operator: corev1.TolerationOpEqual,
					Value:    "true",
					Effect:   corev1.TaintEffectNoExecute,
				},
			}

			By("applying the updated manifest")
			err = applyManifest(&admission, kacConfig.namespace)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating both old and new tolerations are present")
			validateBothTolerations := func() error {
				cmd := exec.Command("kubectl", "get", "deployment", "falcon-kac",
					"-n", kacConfig.namespace,
					"-o", "jsonpath={.spec.template.spec.tolerations}")
				updatedTolerations, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())

				tolerationsStr := string(updatedTolerations)
				if !strings.Contains(tolerationsStr, "node.kubernetes.io/memory-pressure") {
					return fmt.Errorf("previous toleration not preserved: %s", tolerationsStr)
				}
				if !strings.Contains(tolerationsStr, "custom-taint") {
					return fmt.Errorf("new toleration not found: %s", tolerationsStr)
				}

				// Verify we have at least the tolerations we specified
				// (may have additional system-added ones)
				if len(currentTolerations) > 0 && len(updatedTolerations) < len(currentTolerations) {
					return fmt.Errorf("tolerations count decreased unexpectedly")
				}
				return nil
			}
			EventuallyWithOffset(1, validateBothTolerations, defaultTimeout, defaultPollPeriod).Should(Succeed())

			if reconcileLoopCheck {
				By("validating no reconcile loop after toleration changes")
				validateNoReconcileLoop(controllerPodName, namespace, kacConfig.kind, reconcileLoopValidationDuration)
			}
		})

		It("should replace toleration when Key+Effect match but Value/Operator differ", func() {
			By("loading and modifying the FalconAdmission manifest with initial toleration")
			var admission falconv1alpha1.FalconAdmission
			err := loadManifest(manifest, &admission)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			// Set initial toleration with Exists operator
			admission.Spec.AdmissionConfig.Tolerations = []corev1.Toleration{
				{
					Key:      "environment",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
			}

			By("applying the manifest with initial toleration")
			err = applyManifest(&admission, kacConfig.namespace)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			if reconcileLoopCheck {
				By("validating no reconcile loop after initial tolerations")
				validateNoReconcileLoop(controllerPodName, namespace, kacConfig.kind, reconcileLoopValidationDuration)
			}

			By("validating initial toleration with Exists operator")
			validateInitialToleration := func() error {
				cmd := exec.Command("kubectl", "get", "deployment", "falcon-kac",
					"-n", kacConfig.namespace,
					"-o", "jsonpath={.spec.template.spec.tolerations[?(@.key=='environment')].operator}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "Exists") {
					return fmt.Errorf("expected Exists operator not found: %s", output)
				}
				return nil
			}
			EventuallyWithOffset(1, validateInitialToleration, defaultTimeout, defaultPollPeriod).Should(Succeed())

			By("updating toleration with same Key+Effect but different Operator and Value")
			err = loadManifest(manifest, &admission)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			// Update to Equal operator with value for same Key+Effect
			admission.Spec.AdmissionConfig.Tolerations = []corev1.Toleration{
				{
					Key:      "environment",
					Operator: corev1.TolerationOpEqual,
					Value:    "production",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			}

			By("applying the updated manifest")
			err = applyManifest(&admission, kacConfig.namespace)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			if reconcileLoopCheck {
				By("validating no reconcile loop after updating tolerations")
				validateNoReconcileLoop(controllerPodName, namespace, kacConfig.kind, reconcileLoopValidationDuration)
			}

			By("validating toleration was replaced with new Value and Operator")
			validateReplacedToleration := func() error {
				// Check for new operator
				cmd := exec.Command("kubectl", "get", "deployment", "falcon-kac",
					"-n", kacConfig.namespace,
					"-o", "jsonpath={.spec.template.spec.tolerations[?(@.key=='environment')].operator}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "Equal") {
					return fmt.Errorf("expected Equal operator not found: %s", output)
				}

				// Check for new value
				cmd = exec.Command("kubectl", "get", "deployment", "falcon-kac",
					"-n", kacConfig.namespace,
					"-o", "jsonpath={.spec.template.spec.tolerations[?(@.key=='environment')].value}")
				output, err = utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(output), "production") {
					return fmt.Errorf("expected value 'production' not found: %s", output)
				}

				// Verify only one toleration with key 'environment' exists
				cmd = exec.Command("kubectl", "get", "deployment", "falcon-kac",
					"-n", kacConfig.namespace,
					"-o", "jsonpath={.spec.template.spec.tolerations[?(@.key=='environment')]}")
				output, err = utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				// Should only have one match (not duplicate)
				matches := strings.Count(string(output), `"key":"environment"`)
				if matches > 1 {
					return fmt.Errorf("found %d tolerations with key 'environment', expected 1: %s", matches, output)
				}
				return nil
			}
			EventuallyWithOffset(1, validateReplacedToleration, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})

		It("should allow multiple tolerations with same Key but different Effect", func() {
			By("loading and modifying the FalconAdmission manifest with multiple tolerations")
			var admission falconv1alpha1.FalconAdmission
			err := loadManifest(manifest, &admission)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			// Set multiple tolerations with same Key but different Effects
			admission.Spec.AdmissionConfig.Tolerations = []corev1.Toleration{
				{
					Key:      "node-type",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
				{
					Key:      "node-type",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoExecute,
				},
			}

			By("applying the manifest with multiple tolerations")
			err = applyManifest(&admission, kacConfig.namespace)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			if reconcileLoopCheck {
				By("validating no reconcile loop after adding multiple tolerations")
				validateNoReconcileLoop(controllerPodName, namespace, kacConfig.kind, reconcileLoopValidationDuration)
			}

			By("validating both tolerations with same Key but different Effect exist")
			validateBothEffects := func() error {
				// Get all tolerations with key 'node-type'
				cmd := exec.Command("kubectl", "get", "deployment", "falcon-kac",
					"-n", kacConfig.namespace,
					"-o", "jsonpath={.spec.template.spec.tolerations[?(@.key=='node-type')]}")
				output, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())

				// Verify both effects are present
				outputStr := string(output)
				if !strings.Contains(outputStr, "NoSchedule") {
					return fmt.Errorf("NoSchedule effect not found for key 'node-type': %s", outputStr)
				}
				if !strings.Contains(outputStr, "NoExecute") {
					return fmt.Errorf("NoExecute effect not found for key 'node-type': %s", outputStr)
				}

				// Verify we have exactly 2 tolerations with this key (one for each effect)
				effectCount := strings.Count(outputStr, `"effect":`)
				if effectCount != 2 {
					return fmt.Errorf("expected 2 tolerations with key 'node-type', found %d: %s", effectCount, outputStr)
				}
				return nil
			}
			EventuallyWithOffset(1, validateBothEffects, defaultTimeout, defaultPollPeriod).Should(Succeed())
		})

		It("should cleanup successfully", func() {
			kacConfig.manageCrdInstance(crDelete, manifest)
			kacConfig.validateRunningStatus(shouldBeTerminated)
			kacConfig.waitForNamespaceDeletion()
		})
	})
})
