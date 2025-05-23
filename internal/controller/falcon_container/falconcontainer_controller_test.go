package falcon

import (
	"context"
	"fmt"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/internal/controller/common/sensorversion"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("FalconContainer controller", func() {
	Context("FalconContainer controller test", func() {

		const SidecarSensorName = "test-falconsidecarsensor"
		const SidecarSensorNamespace = "falcon-system"
		// The namespaceCounter is a way to create a unique namespace for each test. Namespaces in tests are not reusable.
		// ref: https://book.kubebuilder.io/reference/envtest.html#namespace-usage-limitation
		namespaceCounter := 0
		var testNamespace corev1.Namespace
		var containerNamespacedName types.NamespacedName

		containerImage := "example.com/image:test"
		falconCID := "1234567890ABCDEF1234567890ABCDEF-12"
		ctx := context.Background()

		BeforeEach(func() {
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
			// be aware of the current delete namespace limitations. More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Cleaning up previously used Namespace and shared resources")

			// Delete all deployments
			deployList := &appsv1.DeploymentList{}
			Expect(k8sClient.List(ctx, deployList, client.InNamespace(testNamespace.Namespace))).To(Succeed())
			for _, item := range deployList.Items {
				Expect(k8sClient.Delete(ctx, &item)).To(Succeed())
			}

			Eventually(func() int {
				deployList := &appsv1.DeploymentList{}
				_ = k8sClient.List(ctx, deployList, client.InNamespace(testNamespace.Namespace))
				return len(deployList.Items)
			}, 6*time.Second, 2*time.Second).Should(Equal(0))

			// Delete cluster level resources
			clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: injectorClusterRoleBindingName}, clusterRoleBinding)).To(Succeed())
			Expect(k8sClient.Delete(ctx, clusterRoleBinding)).To(Succeed())

			// Delete FalconContainer custom resource
			falconContainerCR := &falconv1alpha1.FalconContainer{}
			Expect(k8sClient.Get(ctx, containerNamespacedName, falconContainerCR)).To(Succeed())

			// Remove finalizer for successful FalconNodeSensor CR deletion
			patch := client.MergeFrom(falconContainerCR.DeepCopy())
			falconContainerCR.SetFinalizers(nil)
			_ = k8sClient.Patch(ctx, falconContainerCR, patch)

			Expect(k8sClient.Delete(ctx, falconContainerCR)).To(Succeed())

			Eventually(func() bool {
				falconContainerCR := &falconv1alpha1.FalconContainer{}
				err := k8sClient.Get(ctx, containerNamespacedName, falconContainerCR)
				return errors.IsNotFound(err)
			}, 6*time.Second, 2*time.Second).Should(BeTrue())

			_ = k8sClient.Delete(ctx, &testNamespace)
		})

		It("should successfully reconcile a custom resource for FalconContainer", func() {
			By("Creating the custom resource for the Kind FalconContainer")
			falconContainer := &falconv1alpha1.FalconContainer{}
			err := k8sClient.Get(ctx, containerNamespacedName, falconContainer)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				falconContainer := &falconv1alpha1.FalconContainer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      SidecarSensorName,
						Namespace: testNamespace.Name,
					},
					Spec: falconv1alpha1.FalconContainerSpec{
						Falcon: falconv1alpha1.FalconSensor{
							CID: &falconCID,
						},
						Image: &containerImage,
						Registry: falconv1alpha1.RegistrySpec{
							Type: "crowdstrike",
						},
					},
				}

				err = k8sClient.Create(ctx, falconContainer)
				Expect(err).To(Not(HaveOccurred()))
			}

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconContainer{}
				return k8sClient.Get(ctx, containerNamespacedName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
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

			_, err = falconContainerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: containerNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			// TODO: serviceAccount reconciliation might be removed in the future
			_, err = falconContainerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: containerNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			// TODO: clusterRoleBinding reconciliation might be removed in the future
			_, err = falconContainerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: containerNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if TLS Secret was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.Secret{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: "falcon-sidecar-injector-tls", Namespace: SidecarSensorNamespace}, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking if ConfigMap was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.ConfigMap{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: "falcon-sidecar-injector-config", Namespace: SidecarSensorNamespace}, found)
			}, time.Minute, time.Second).Should(Succeed())

			_, err = falconContainerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: containerNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func() error {
				found := &appsv1.Deployment{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: "falcon-sidecar-injector", Namespace: SidecarSensorNamespace}, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking if Service was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.Service{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: "falcon-sidecar-injector", Namespace: SidecarSensorNamespace}, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking if pods were successfully created in the reconciliation")
			Eventually(func() error {
				pod, err := k8sutils.GetReadyPod(k8sClient, ctx, SidecarSensorNamespace, map[string]string{common.FalconComponentKey: common.FalconSidecarSensor})
				if err != nil && err != k8sutils.ErrNoWebhookServicePodReady {
					return err
				}
				if pod.Name == "" {
					_, err = falconContainerReconciler.Reconcile(ctx, reconcile.Request{
						NamespacedName: containerNamespacedName,
					})
				}
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking the latest Status Condition added to the FalconContainer instance")
			Eventually(func() error {
				if len(falconContainer.Status.Conditions) != 0 {
					latestStatusCondition := falconContainer.Status.Conditions[len(falconContainer.Status.Conditions)-1]
					expectedLatestStatusCondition := metav1.Condition{Type: falconv1alpha1.ConditionDeploymentReady,
						Status: metav1.ConditionTrue, Reason: falconv1alpha1.ReasonInstallSucceeded,
						Message: "FalconContainer installation completed"}
					if latestStatusCondition != expectedLatestStatusCondition {
						return fmt.Errorf("The latest status condition added to the FalconContainer instance is not as expected")
					}
				}
				return nil
			}, time.Minute, time.Second).Should(Succeed())
		})

		It("should correctly handle and inject existing secrets into configmap", func() {
			By("Creating test secrets")
			clientId := "test-client-id"
			clientSecret := "test-client-secret"
			provisioningToken := "1a2b3c4d"
			secretName := "falcon-secret"
			testSecretNamespace := "falcon-secret"

			falconSecretNamespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testSecretNamespace,
					Namespace: testSecretNamespace,
				},
			}

			err := k8sClient.Create(ctx, &falconSecretNamespace)
			Expect(err).To(Not(HaveOccurred()))

			testSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: testSecretNamespace,
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"falcon-client-id":          clientId,
					"falcon-client-secret":      clientSecret,
					"falcon-cid":                falconCID,
					"falcon-provisioning-token": provisioningToken,
				},
			}
			err = k8sClient.Create(ctx, testSecret)
			Expect(err).To(Not(HaveOccurred()))

			By("Creating the FalconAdmission CR with FalconSecret configured")
			falconContainer := &falconv1alpha1.FalconContainer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SidecarSensorName,
					Namespace: containerNamespacedName.Name,
				},
				Spec: falconv1alpha1.FalconContainerSpec{
					InstallNamespace: containerNamespacedName.Namespace,
					FalconSecret: falconv1alpha1.FalconSecret{
						Enabled:    true,
						Namespace:  testSecretNamespace,
						SecretName: secretName,
					},
					Falcon: falconv1alpha1.FalconSensor{
						CID: &falconCID,
					},
					Image: &containerImage,
					Registry: falconv1alpha1.RegistrySpec{
						Type: "crowdstrike",
					},
				},
			}

			err = k8sClient.Create(ctx, falconContainer)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				falconContainer := &falconv1alpha1.FalconContainer{}
				return k8sClient.Get(ctx, containerNamespacedName, falconContainer)
			}, 6*time.Second, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			tracker, cancel := sensorversion.NewTestTracker()
			defer cancel()

			falconContainerReconciler := &FalconContainerReconciler{
				Client:  k8sClient,
				Reader:  k8sReader,
				Scheme:  k8sClient.Scheme(),
				tracker: tracker,
			}

			for range 4 {
				_, err = falconContainerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: containerNamespacedName,
				})
				Expect(err).To(Not(HaveOccurred()))
			}

			By("Checking if Deployment was created with updated config map")
			Eventually(func() error {
				containerConfigMap := &corev1.ConfigMap{}
				err = common.GetNamespacedObject(
					ctx,
					falconContainerReconciler.Client,
					falconContainerReconciler.Reader,
					types.NamespacedName{Name: injectorConfigMapName, Namespace: containerNamespacedName.Namespace},
					containerConfigMap,
				)
				if err != nil {
					return fmt.Errorf("failed to get container configmap: %w", err)
				}

				containerDeployment := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      injectorName,
					Namespace: containerNamespacedName.Namespace,
				}, containerDeployment)
				if err != nil {
					return fmt.Errorf("failed to get container deployment: %w", err)
				}

				// Check for environment variables with secret references
				containers := containerDeployment.Spec.Template.Spec.Containers
				if len(containers) == 0 {
					return fmt.Errorf("no containers found in container deployment")
				}

				Expect(containerDeployment).To(haveContainerNamed("falcon-sensor"))
				Expect(containerDeployment).To(haveContainerWithConfigMapEnvFrom("falcon-sensor", injectorConfigMapName))
				Expect(containerConfigMap.Data["FALCONCTL_OPT_CID"]).To(Equal(falconCID))
				Expect(containerConfigMap.Data["FALCONCTL_OPT_PROVISIONING_TOKEN"]).To(Equal(provisioningToken))

				return nil
			}, 10*time.Second, time.Second).Should(Succeed())

			By("Cleaning up the test specific resources")
			err = k8sClient.Delete(ctx, testSecret)
			Expect(err).To(Not(HaveOccurred()))
		})
	})
})
