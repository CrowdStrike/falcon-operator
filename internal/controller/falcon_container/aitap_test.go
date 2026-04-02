package falcon

import (
	"context"
	"fmt"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/common/sensorversion"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("FalconContainer AITap reconciliation", func() {
	Context("AITap secret creation", func() {
		const SidecarSensorName = "test-aitap-sensor"
		const SidecarSensorNamespace = "falcon-aitap-test"
		namespaceCounter := 0
		var testNamespace corev1.Namespace
		var containerNamespacedName types.NamespacedName
		var ctx context.Context

		containerImage := "example.com/image:test"
		falconCID := "1234567890ABCDEF1234567890ABCDEF-12"
		aidrToken := "test-aidr-token-123"

		BeforeEach(func() {
			ctx = context.Background()
			namespaceCounter += 1
			currentNamespaceString := fmt.Sprintf("%s-%d", SidecarSensorNamespace, namespaceCounter)
			testNamespace = corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      currentNamespaceString,
					Namespace: currentNamespaceString,
				},
			}

			containerNamespacedName = types.NamespacedName{Name: SidecarSensorName, Namespace: currentNamespaceString}

			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, &testNamespace)
			Expect(err).To(Not(HaveOccurred()))
		})

		AfterEach(func() {
			By("Cleaning up previously used Namespace and resources")

			// Delete FalconContainer custom resource
			falconContainerCR := &falconv1alpha1.FalconContainer{}
			if err := k8sClient.Get(ctx, containerNamespacedName, falconContainerCR); err == nil {
				Expect(k8sClient.Delete(ctx, falconContainerCR)).To(Succeed())

				Eventually(func() bool {
					falconContainerCR := &falconv1alpha1.FalconContainer{}
					err := k8sClient.Get(ctx, containerNamespacedName, falconContainerCR)
					return errors.IsNotFound(err)
				}, 6*time.Second, 2*time.Second).Should(BeTrue())
			}

			_ = k8sClient.Delete(ctx, &testNamespace)
		})

		It("should create AITap AI-DR secret in specific namespaces", func() {
			By("Creating additional namespaces for AITap deployment")
			app1Namespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "app1",
				},
			}
			app2Namespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "app2",
				},
			}

			err := k8sClient.Create(ctx, &app1Namespace)
			Expect(err).To(Not(HaveOccurred()))
			err = k8sClient.Create(ctx, &app2Namespace)
			Expect(err).To(Not(HaveOccurred()))

			By("Creating FalconContainer with AITap configuration")
			falconContainer := &falconv1alpha1.FalconContainer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SidecarSensorName,
					Namespace: testNamespace.Name,
				},
				Spec: falconv1alpha1.FalconContainerSpec{
					InstallNamespace: testNamespace.Name,
					Falcon: falconv1alpha1.FalconSensor{
						CID: &falconCID,
					},
					Image: &containerImage,
					Registry: falconv1alpha1.RegistrySpec{
						Type: "crowdstrike",
					},
					Injector: falconv1alpha1.FalconContainerInjectorSpec{
						AITap: falconv1alpha1.AITapSpec{
							AidrCollectorApiToken:   aidrToken,
							AidrCollectorBaseApiUrl: "https://api.example.com",
							Namespaces:              "app1,app2",
						},
					},
				},
			}

			err = k8sClient.Create(ctx, falconContainer)
			Expect(err).To(Not(HaveOccurred()))

			By("Reconciling the FalconContainer resource")
			tracker, cancel := sensorversion.NewTestTracker()
			defer cancel()

			falconContainerReconciler := &FalconContainerReconciler{
				Client:  k8sClient,
				Reader:  k8sReader,
				Scheme:  k8sClient.Scheme(),
				tracker: tracker,
			}

			_, err = falconContainerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: containerNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			secretName := "falcon-aitap-aidr-secret"

			By("Verifying AITap secret does NOT exist in install namespace")
			Consistently(func() bool {
				secret := &corev1.Secret{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      secretName,
					Namespace: testNamespace.Name,
				}, secret)
				return errors.IsNotFound(err)
			}, 3*time.Second, time.Second).Should(BeTrue(), "Secret should not be created in install namespace")

			By("Verifying AITap secret exists in app1 namespace")
			Eventually(func() error {
				secret := &corev1.Secret{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      secretName,
					Namespace: "app1",
				}, secret)
			}, 10*time.Second, time.Second).Should(Succeed())

			By("Verifying AITap secret exists in app2 namespace")
			Eventually(func() error {
				secret := &corev1.Secret{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      secretName,
					Namespace: "app2",
				}, secret)
			}, 10*time.Second, time.Second).Should(Succeed())

			By("Cleaning up test namespaces")
			_ = k8sClient.Delete(ctx, &app1Namespace)
			_ = k8sClient.Delete(ctx, &app2Namespace)
		})

		It("should configure AITap environment variables in ConfigMap", func() {
			By("Creating FalconContainer with AITap configuration including optional base URL")
			falconContainer := &falconv1alpha1.FalconContainer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SidecarSensorName,
					Namespace: testNamespace.Name,
				},
				Spec: falconv1alpha1.FalconContainerSpec{
					InstallNamespace: testNamespace.Name,
					Falcon: falconv1alpha1.FalconSensor{
						CID: &falconCID,
					},
					Image: &containerImage,
					Registry: falconv1alpha1.RegistrySpec{
						Type: "crowdstrike",
					},
					Injector: falconv1alpha1.FalconContainerInjectorSpec{
						AITap: falconv1alpha1.AITapSpec{
							AidrCollectorApiToken:   aidrToken,
							AidrCollectorBaseApiUrl: "https://custom-api.example.com",
							Namespaces:              "app1,app2",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, falconContainer)
			Expect(err).To(Not(HaveOccurred()))

			By("Reconciling the FalconContainer resource")
			tracker, cancel := sensorversion.NewTestTracker()
			defer cancel()

			falconContainerReconciler := &FalconContainerReconciler{
				Client:  k8sClient,
				Reader:  k8sReader,
				Scheme:  k8sClient.Scheme(),
				tracker: tracker,
			}

			_, err = falconContainerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: containerNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Verifying ConfigMap contains AITap environment variables")
			configMapName := "falcon-sidecar-injector-config"
			Eventually(func() error {
				configMap := &corev1.ConfigMap{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      configMapName,
					Namespace: testNamespace.Name,
				}, configMap)
				if err != nil {
					return err
				}

				// Verify AITap environment variables
				secretName := "falcon-aitap-aidr-secret"
				if configMap.Data["FALCON_AITAP_AIDR_SECRET_NAME"] != secretName {
					return fmt.Errorf("expected FALCON_AITAP_AIDR_SECRET_NAME=%s, got %s",
						secretName, configMap.Data["FALCON_AITAP_AIDR_SECRET_NAME"])
				}

				if configMap.Data["FALCON_AITAP_AIDR_COLLECTOR_BASE_API_URL"] != "https://custom-api.example.com" {
					return fmt.Errorf("expected FALCON_AITAP_AIDR_COLLECTOR_BASE_API_URL=https://custom-api.example.com, got %s",
						configMap.Data["FALCON_AITAP_AIDR_COLLECTOR_BASE_API_URL"])
				}

				if configMap.Data["FALCON_AITAP_NAMESPACES"] != "app1,app2" {
					return fmt.Errorf("expected FALCON_AITAP_NAMESPACES=app1,app2, got %s",
						configMap.Data["FALCON_AITAP_NAMESPACES"])
				}

				return nil
			}, 10*time.Second, time.Second).Should(Succeed())
		})

		It("should skip AITap secret creation when UseExistingSecret is enabled", func() {
			ctx := context.Background()
			testLog := logr.Discard()
			falconContainer := &falconv1alpha1.FalconContainer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-fc",
					Namespace: testNamespace.Name,
				},
				Spec: falconv1alpha1.FalconContainerSpec{
					InstallNamespace: testNamespace.Name,
					Injector: falconv1alpha1.FalconContainerInjectorSpec{
						AITap: falconv1alpha1.AITapSpec{
							AidrCollectorApiToken:   "test-token-disabled",
							AidrSecretName:          "externally-managed-secret",
							AidrCollectorBaseApiUrl: "https://api.example.com",
							Namespaces:              "app1,app2",
							UseExistingSecret:       true,
						},
					},
				},
			}

			tracker, cancel := sensorversion.NewTestTracker()
			defer cancel()

			falconContainerReconciler := &FalconContainerReconciler{
				Client:  k8sClient,
				Reader:  k8sReader,
				Scheme:  k8sClient.Scheme(),
				tracker: tracker,
			}

			// Reconcile AITap secrets
			secretList, err := falconContainerReconciler.reconcileAITapSecrets(ctx, testLog, falconContainer)
			Expect(err).ToNot(HaveOccurred())
			Expect(secretList.Items).To(BeEmpty(), "No secrets should be created when UseExistingSecret is true")

			// Verify no secrets were created in any namespace
			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      "externally-managed-secret",
				Namespace: testNamespace.Name,
			}, secret)
			Expect(errors.IsNotFound(err)).To(BeTrue(), "Secret should not exist in install namespace")
		})

		It("should configure AITap environment variables in ConfigMap when UseExistingSecret is enabled", func() {
			ctx := context.Background()
			testLog := logr.Discard()
			listenPort := int32(4433)
			falconContainer := &falconv1alpha1.FalconContainer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-fc-external",
					Namespace: testNamespace.Name,
				},
				Spec: falconv1alpha1.FalconContainerSpec{
					InstallNamespace: testNamespace.Name,
					Falcon: falconv1alpha1.FalconSensor{
						CID: &falconCID,
					},
					Image: &containerImage,
					Registry: falconv1alpha1.RegistrySpec{
						Type: "crowdstrike",
					},
					Injector: falconv1alpha1.FalconContainerInjectorSpec{
						ListenPort: &listenPort,
						AITap: falconv1alpha1.AITapSpec{
							AidrSecretName:          "my-external-secret",
							AidrCollectorBaseApiUrl: "https://api.example.com",
							Namespaces:              "ns1,ns2,ns3",
							UseExistingSecret:       true,
						},
					},
				},
			}

			tracker, cancel := sensorversion.NewTestTracker()
			defer cancel()

			falconContainerReconciler := &FalconContainerReconciler{
				Client:  k8sClient,
				Reader:  k8sReader,
				Scheme:  k8sClient.Scheme(),
				tracker: tracker,
			}

			configMap, err := falconContainerReconciler.newConfigMap(ctx, testLog, falconContainer)
			Expect(err).ToNot(HaveOccurred())

			// Verify AITap environment variables are set even without token
			Expect(configMap.Data).To(HaveKeyWithValue("FALCON_AITAP_AIDR_SECRET_NAME", "my-external-secret"))
			Expect(configMap.Data).To(HaveKeyWithValue("FALCON_AITAP_AIDR_COLLECTOR_BASE_API_URL", "https://api.example.com"))
			Expect(configMap.Data).To(HaveKeyWithValue("FALCON_AITAP_NAMESPACES", "ns1,ns2,ns3"))
		})

		It("should return validation errors for invalid AITap configs", func() {
			cases := []struct {
				name        string
				aitap       falconv1alpha1.AITapSpec
				expectedMsg string
			}{
				{
					name: "namespaces and allNamespaces both set",
					aitap: falconv1alpha1.AITapSpec{
						AidrCollectorApiToken:   "token",
						AidrCollectorBaseApiUrl: "https://api.example.com",
						Namespaces:              "app1",
						AllNamespaces:           true,
					},
					expectedMsg: "'namespaces' and 'allNamespaces' cannot both be set",
				},
				{
					name: "aidrCollectorBaseApiUrl missing with namespaces set",
					aitap: falconv1alpha1.AITapSpec{
						AidrCollectorApiToken: "token",
						Namespaces:            "app1",
					},
					expectedMsg: "'aidrCollectorBaseApiUrl' is required",
				},
				{
					name: "aidrCollectorApiToken missing and useExistingSecret false",
					aitap: falconv1alpha1.AITapSpec{
						AidrCollectorBaseApiUrl: "https://api.example.com",
						Namespaces:              "app1",
					},
					expectedMsg: "'aidrCollectorApiToken' is required",
				},
				{
					name: "useExistingSecret true but aidrSecretName empty",
					aitap: falconv1alpha1.AITapSpec{
						AidrCollectorBaseApiUrl: "https://api.example.com",
						Namespaces:              "app1",
						UseExistingSecret:       true,
					},
					expectedMsg: "'aidrSecretName' is required when 'useExistingSecret' is true",
				},
			}

			tracker, cancel := sensorversion.NewTestTracker()
			defer cancel()
			r := &FalconContainerReconciler{
				Client:  k8sClient,
				Reader:  k8sReader,
				Scheme:  k8sClient.Scheme(),
				tracker: tracker,
			}

			for _, tc := range cases {
				falconContainer := &falconv1alpha1.FalconContainer{
					Spec: falconv1alpha1.FalconContainerSpec{
						InstallNamespace: testNamespace.Name,
						Injector: falconv1alpha1.FalconContainerInjectorSpec{
							AITap: tc.aitap,
						},
					},
				}
				_, err := r.reconcileAITapSecrets(context.Background(), logr.Discard(), falconContainer)
				Expect(err).To(HaveOccurred(), tc.name)
				Expect(err.Error()).To(ContainSubstring(tc.expectedMsg), tc.name)
			}
		})

		It("should exclude install namespace and reserved namespaces from allNamespaces propagation", func() {
			ctx := context.Background()

			for _, name := range []string{"falcon-kac", "falcon-iar", "app1", "openshift-monitoring"} {
				ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
				_ = k8sClient.Create(ctx, &ns)
				nsCopy := ns
				defer k8sClient.Delete(ctx, &nsCopy) //nolint:errcheck
			}

			falconContainer := &falconv1alpha1.FalconContainer{
				Spec: falconv1alpha1.FalconContainerSpec{
					InstallNamespace: testNamespace.Name,
					Injector: falconv1alpha1.FalconContainerInjectorSpec{
						AITap: falconv1alpha1.AITapSpec{
							AidrCollectorApiToken:   aidrToken,
							AidrCollectorBaseApiUrl: "https://api.example.com",
							AllNamespaces:           true,
						},
					},
				},
			}

			tracker, cancel := sensorversion.NewTestTracker()
			defer cancel()

			r := &FalconContainerReconciler{
				Client:  k8sClient,
				Reader:  k8sReader,
				Scheme:  k8sClient.Scheme(),
				tracker: tracker,
			}

			namespaces, err := r.getAITapTargetNamespaces(ctx, logr.Discard(), falconContainer)
			Expect(err).ToNot(HaveOccurred())
			Expect(namespaces).NotTo(ContainElement(testNamespace.Name))
			Expect(namespaces).NotTo(ContainElement("falcon-system"))
			Expect(namespaces).NotTo(ContainElement("falcon-kac"))
			Expect(namespaces).NotTo(ContainElement("falcon-iar"))
			Expect(namespaces).NotTo(ContainElement("openshift-monitoring"))
			Expect(namespaces).To(ContainElement("app1"))
		})
	})
})
