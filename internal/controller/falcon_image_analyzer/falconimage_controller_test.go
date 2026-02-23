package falcon

import (
	"context"
	"fmt"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
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
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("FalconImageAnalyzer controller", func() {
	Context("FalconImageAnalyzer controller test", func() {

		const ImageAnalyzerName = "test-falconimageanalyzer"
		const ImageAnalyzerNamespace = "falcon-image-analyzer"
		// The namespaceCounter is a way to create a unique namespace for each test. Namespaces in tests are not reusable.
		// ref: https://book.kubebuilder.io/reference/envtest.html#namespace-usage-limitation
		namespaceCounter := 0
		var testNamespace corev1.Namespace
		var imageAnalyzerNamespacedName types.NamespacedName

		imageAnalyzerImage := "example.com/image:test"
		falconCID := "1234567890ABCDEF1234567890ABCDEF-12"
		ctx := context.Background()

		BeforeEach(func() {
			namespaceCounter += 1
			currentNamespaceString := fmt.Sprintf("%s-%d", ImageAnalyzerNamespace, namespaceCounter)
			testNamespace = corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      currentNamespaceString,
					Namespace: currentNamespaceString,
				},
			}

			imageAnalyzerNamespacedName = types.NamespacedName{Name: ImageAnalyzerName, Namespace: currentNamespaceString}

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
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: imageClusterRoleBindingName}, clusterRoleBinding); err == nil {
				Expect(k8sClient.Delete(ctx, clusterRoleBinding)).To(Succeed())
			}

			// Delete FalconImageAnalyzer custom resource
			falconImageAnalyzerCR := &falconv1alpha1.FalconImageAnalyzer{}
			Expect(k8sClient.Get(ctx, imageAnalyzerNamespacedName, falconImageAnalyzerCR)).To(Succeed())

			Expect(k8sClient.Delete(ctx, falconImageAnalyzerCR)).To(Succeed())

			Eventually(func() bool {
				falconImageAnalyzerCR := &falconv1alpha1.FalconImageAnalyzer{}
				err := k8sClient.Get(ctx, imageAnalyzerNamespacedName, falconImageAnalyzerCR)
				return errors.IsNotFound(err)
			}, 6*time.Second, 2*time.Second).Should(BeTrue())

			_ = k8sClient.Delete(ctx, &testNamespace)
		})

		It("should successfully reconcile a custom resource for FalconImageAnalyzer", func() {
			By("Creating the custom resource for the Kind FalconImageAnalyzer")
			falconImageAnalyzer := &falconv1alpha1.FalconImageAnalyzer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ImageAnalyzerName,
					Namespace: testNamespace.Name,
				},
				Spec: falconv1alpha1.FalconImageAnalyzerSpec{
					InstallNamespace: imageAnalyzerNamespacedName.Namespace,
					FalconAPI: &falconv1alpha1.FalconAPI{
						CID:         &falconCID,
						CloudRegion: "autodiscover",
					},
					Image: imageAnalyzerImage,
					Registry: falconv1alpha1.RegistrySpec{
						Type: "crowdstrike",
					},
				},
			}

			err := k8sClient.Create(ctx, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconImageAnalyzer{}
				return k8sClient.Get(ctx, imageAnalyzerNamespacedName, found)
			}, 6*time.Second, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			falconImageAnalyzerReconciler := &FalconImageAnalyzerReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			_, err = falconImageAnalyzerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: imageAnalyzerNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if Service Account was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.ServiceAccount{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: common.ImageServiceAccountName, Namespace: imageAnalyzerNamespacedName.Namespace}, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Checking if ConfigMap was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.ConfigMap{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: falconImageAnalyzer.Name + "-config", Namespace: imageAnalyzerNamespacedName.Namespace}, found)
			}, 6*time.Second, time.Second).Should(Succeed())

			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func() error {
				found := &appsv1.Deployment{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: ImageAnalyzerName, Namespace: imageAnalyzerNamespacedName.Namespace}, found)
			}, 6*time.Second, time.Second).Should(Succeed())

			//TODO: Revisit test setup for pods to be created successfully
			//By("Skip - Checking if pod was successfully created in the reconciliation")
			//Eventually(func() error {
			//	_, err := k8sutils.GetReadyPod(k8sClient, ctx, imageAnalyzerNamespacedName.Namespace, map[string]string{common.FalconComponentKey: common.FalconImageAnalyzer})
			//
			//	return err
			//}, 20*time.Second, time.Second).Should(Succeed())

			By("Checking the latest Status Condition added to the FalconImageAnalyzer instance")
			Eventually(func() error {
				if len(falconImageAnalyzer.Status.Conditions) != 0 {
					latestStatusCondition := falconImageAnalyzer.Status.Conditions[len(falconImageAnalyzer.Status.Conditions)-1]
					expectedLatestStatusCondition := metav1.Condition{Type: falconv1alpha1.ConditionDeploymentReady,
						Status: metav1.ConditionTrue, Reason: falconv1alpha1.ReasonInstallSucceeded,
						Message: "FalconImageAnalyzer installation completed"}
					if latestStatusCondition != expectedLatestStatusCondition {
						return fmt.Errorf("The latest status condition added to the FalconImageAnalyzer instance is not as expected")
					}
				}
				return nil
			}, 6*time.Second, time.Second).Should(Succeed())
		})

		It("should correctly handle and inject existing secrets into configmap", func() {
			By("Creating test secrets")
			clientId := "test-client-id"
			clientSecret := "test-client-secret"
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
					"falcon-client-id":     clientId,
					"falcon-client-secret": clientSecret,
					"falcon-cid":           falconCID,
				},
			}
			err = k8sClient.Create(ctx, testSecret)
			Expect(err).To(Not(HaveOccurred()))

			By("Creating the FalconImageAnalyzer CR with FalconSecret configured")
			falconImageAnalyzer := &falconv1alpha1.FalconImageAnalyzer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ImageAnalyzerName,
					Namespace: imageAnalyzerNamespacedName.Name,
				},
				Spec: falconv1alpha1.FalconImageAnalyzerSpec{
					InstallNamespace: imageAnalyzerNamespacedName.Namespace,
					FalconAPI: &falconv1alpha1.FalconAPI{
						CID:         &falconCID,
						CloudRegion: "autodiscover",
					},
					FalconSecret: falconv1alpha1.FalconSecret{
						Enabled:    true,
						Namespace:  testSecretNamespace,
						SecretName: secretName,
					},
					Image: imageAnalyzerImage,
					Registry: falconv1alpha1.RegistrySpec{
						Type: "crowdstrike",
					},
				},
			}

			err = k8sClient.Create(ctx, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				falconImageAnalyzer := &falconv1alpha1.FalconImageAnalyzer{}
				return k8sClient.Get(ctx, imageAnalyzerNamespacedName, falconImageAnalyzer)
			}, 6*time.Second, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			falconImageAnalyzerReconciler := &FalconImageAnalyzerReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			_, err = falconImageAnalyzerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: imageAnalyzerNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if Deployment was created with updated config map")
			Eventually(func() error {
				imageAnalyzerConfigMap := &corev1.ConfigMap{}
				err = common.GetNamespacedObject(
					ctx,
					falconImageAnalyzerReconciler.Client,
					falconImageAnalyzerReconciler.Reader,
					types.NamespacedName{
						Name:      ImageAnalyzerName + "-config",
						Namespace: imageAnalyzerNamespacedName.Namespace,
					},
					imageAnalyzerConfigMap,
				)
				if err != nil {
					return fmt.Errorf("failed to get imageAnalyzer configmap: %w", err)
				}

				imageAnalyzerDeployment := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      ImageAnalyzerName,
					Namespace: imageAnalyzerNamespacedName.Namespace,
				}, imageAnalyzerDeployment)
				if err != nil {
					return fmt.Errorf("failed to get imageAnalyzer deployment: %w", err)
				}

				// Check for environment variables with secret references
				imageAnalyzers := imageAnalyzerDeployment.Spec.Template.Spec.Containers
				if len(imageAnalyzers) == 0 {
					return fmt.Errorf("no imageAnalyzers found in imageAnalyzer deployment")
				}

				Expect(imageAnalyzerDeployment).To(haveContainerNamed("falcon-image-analyzer"))
				Expect(imageAnalyzerDeployment).To(haveContainerWithConfigMapEnvFrom("falcon-image-analyzer", falconImageAnalyzer.Name+"-config"))
				Expect(imageAnalyzerConfigMap.Data["AGENT_CID"]).To(Equal(falconCID))
				Expect(imageAnalyzerConfigMap.Data["AGENT_CLIENT_ID"]).To(Equal(clientId))
				Expect(imageAnalyzerConfigMap.Data["AGENT_CLIENT_SECRET"]).To(Equal(clientSecret))

				return nil
			}, 10*time.Second, time.Second).Should(Succeed())

			By("Cleaning up the test specific resources")
			err = k8sClient.Delete(ctx, testSecret)
			Expect(err).To(Not(HaveOccurred()))
		})

		// Testing reconcileServiceAccount return value
		It("should return false when creating a new service account", func() {
			log := zap.New(zap.UseDevMode(true))

			By("Creating the custom resource for the Kind FalconImageAnalyzer")
			falconImageAnalyzer := &falconv1alpha1.FalconImageAnalyzer{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "falcon.crowdstrike.com/v1alpha1",
					Kind:       "FalconImageAnalyzer",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      ImageAnalyzerName,
					Namespace: testNamespace.Name,
				},
				Spec: falconv1alpha1.FalconImageAnalyzerSpec{
					InstallNamespace: imageAnalyzerNamespacedName.Namespace,
					FalconAPI: &falconv1alpha1.FalconAPI{
						ClientId:     "test-client-id",
						ClientSecret: "test-client-secret",
						CloudRegion:  "us-1",
					},
					Registry: falconv1alpha1.RegistrySpec{
						Type: "crowdstrike",
					},
					Image: imageAnalyzerImage,
				},
			}
			err := k8sClient.Create(ctx, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconImageAnalyzer{}
				return k8sClient.Get(ctx, imageAnalyzerNamespacedName, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Creating the reconciler")
			falconImageAnalyzerReconciler := &FalconImageAnalyzerReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			By("Calling reconcileServiceAccount when no service account exists")
			updated, err := falconImageAnalyzerReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: imageAnalyzerNamespacedName,
			}, log, falconImageAnalyzer)

			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse(), "reconcileServiceAccount should return false when creating a new service account")

			By("Verifying the service account was created")
			Eventually(func() error {
				found := &corev1.ServiceAccount{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.ImageServiceAccountName,
					Namespace: testNamespace.Name,
				}, found)
			}, 5*time.Second, time.Second).Should(Succeed())
		})

		It("should return false when only annotations are updated", func() {
			log := zap.New(zap.UseDevMode(true))

			By("Creating the custom resource for the Kind FalconImageAnalyzer")
			falconImageAnalyzer := &falconv1alpha1.FalconImageAnalyzer{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "falcon.crowdstrike.com/v1alpha1",
					Kind:       "FalconImageAnalyzer",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      ImageAnalyzerName,
					Namespace: testNamespace.Name,
				},
				Spec: falconv1alpha1.FalconImageAnalyzerSpec{
					InstallNamespace: imageAnalyzerNamespacedName.Namespace,
					FalconAPI: &falconv1alpha1.FalconAPI{
						ClientId:     "test-client-id",
						ClientSecret: "test-client-secret",
						CloudRegion:  "us-1",
					},
					Registry: falconv1alpha1.RegistrySpec{
						Type: "crowdstrike",
					},
					Image: imageAnalyzerImage,
				},
			}
			err := k8sClient.Create(ctx, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconImageAnalyzer{}
				return k8sClient.Get(ctx, imageAnalyzerNamespacedName, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Creating the reconciler")
			falconImageAnalyzerReconciler := &FalconImageAnalyzerReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			By("Creating initial service account")
			updated, err := falconImageAnalyzerReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: imageAnalyzerNamespacedName,
			}, log, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse())

			By("Updating annotations in FalconImageAnalyzer spec")
			falconImageAnalyzer.Spec.ImageAnalyzerConfig.ServiceAccount.Annotations = map[string]string{
				"new-annotation": "new-value",
			}

			By("Calling reconcileServiceAccount with updated annotations")
			updated, err = falconImageAnalyzerReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: imageAnalyzerNamespacedName,
			}, log, falconImageAnalyzer)

			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse(), "reconcileServiceAccount should return false when only annotations are updated")

			By("Verifying annotations were updated")
			Eventually(func() map[string]string {
				found := &corev1.ServiceAccount{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.ImageServiceAccountName,
					Namespace: testNamespace.Name,
				}, found)
				return found.ObjectMeta.Annotations
			}, 5*time.Second, time.Second).Should(HaveKey("new-annotation"))
		})

		It("should return true when imagePullSecrets are updated", func() {
			log := zap.New(zap.UseDevMode(true))

			By("Creating the custom resource for the Kind FalconImageAnalyzer")
			falconImageAnalyzer := &falconv1alpha1.FalconImageAnalyzer{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "falcon.crowdstrike.com/v1alpha1",
					Kind:       "FalconImageAnalyzer",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      ImageAnalyzerName,
					Namespace: testNamespace.Name,
				},
				Spec: falconv1alpha1.FalconImageAnalyzerSpec{
					InstallNamespace: imageAnalyzerNamespacedName.Namespace,
					FalconAPI: &falconv1alpha1.FalconAPI{
						ClientId:     "test-client-id",
						ClientSecret: "test-client-secret",
						CloudRegion:  "us-1",
					},
					Registry: falconv1alpha1.RegistrySpec{
						Type: "crowdstrike",
					},
					Image: imageAnalyzerImage,
				},
			}
			err := k8sClient.Create(ctx, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconImageAnalyzer{}
				return k8sClient.Get(ctx, imageAnalyzerNamespacedName, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Creating the reconciler")
			falconImageAnalyzerReconciler := &FalconImageAnalyzerReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			By("Creating initial service account with default imagePullSecrets")
			updated, err := falconImageAnalyzerReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: imageAnalyzerNamespacedName,
			}, log, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse())

			By("Verifying initial imagePullSecrets")
			Eventually(func() []corev1.LocalObjectReference {
				found := &corev1.ServiceAccount{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.ImageServiceAccountName,
					Namespace: testNamespace.Name,
				}, found)
				return found.ImagePullSecrets
			}, 5*time.Second, time.Second).Should(Equal([]corev1.LocalObjectReference{
				{Name: common.FalconPullSecretName},
			}))

			By("Adding a new imagePullSecret to FalconImageAnalyzer spec")
			falconImageAnalyzer.Spec.ImageAnalyzerConfig.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: "additional-secret"},
			}

			By("Calling reconcileServiceAccount with updated imagePullSecrets")
			updated, err = falconImageAnalyzerReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: imageAnalyzerNamespacedName,
			}, log, falconImageAnalyzer)

			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeTrue(), "reconcileServiceAccount should return true when imagePullSecrets are updated")

			By("Verifying imagePullSecrets were updated")
			Eventually(func() []corev1.LocalObjectReference {
				found := &corev1.ServiceAccount{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.ImageServiceAccountName,
					Namespace: testNamespace.Name,
				}, found)
				return found.ImagePullSecrets
			}, 5*time.Second, time.Second).Should(Equal([]corev1.LocalObjectReference{
				{Name: common.FalconPullSecretName},
				{Name: "additional-secret"},
			}))
		})

		It("should return false when no changes are made to service account", func() {
			log := zap.New(zap.UseDevMode(true))

			By("Creating the custom resource for the Kind FalconImageAnalyzer")
			falconImageAnalyzer := &falconv1alpha1.FalconImageAnalyzer{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "falcon.crowdstrike.com/v1alpha1",
					Kind:       "FalconImageAnalyzer",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      ImageAnalyzerName,
					Namespace: testNamespace.Name,
				},
				Spec: falconv1alpha1.FalconImageAnalyzerSpec{
					InstallNamespace: imageAnalyzerNamespacedName.Namespace,
					FalconAPI: &falconv1alpha1.FalconAPI{
						ClientId:     "test-client-id",
						ClientSecret: "test-client-secret",
						CloudRegion:  "us-1",
					},
					Registry: falconv1alpha1.RegistrySpec{
						Type: "crowdstrike",
					},
					Image: imageAnalyzerImage,
				},
			}
			err := k8sClient.Create(ctx, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconImageAnalyzer{}
				return k8sClient.Get(ctx, imageAnalyzerNamespacedName, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Creating the reconciler")
			falconImageAnalyzerReconciler := &FalconImageAnalyzerReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			By("Creating initial service account")
			updated, err := falconImageAnalyzerReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: imageAnalyzerNamespacedName,
			}, log, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse())

			By("Calling reconcileServiceAccount again without any changes")
			updated, err = falconImageAnalyzerReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: imageAnalyzerNamespacedName,
			}, log, falconImageAnalyzer)

			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse(), "reconcileServiceAccount should return false when no changes are made")
		})

		It("should return true when imagePullSecrets are removed", func() {
			log := zap.New(zap.UseDevMode(true))

			By("Creating the custom resource for the Kind FalconImageAnalyzer with extra imagePullSecret")
			falconImageAnalyzer := &falconv1alpha1.FalconImageAnalyzer{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "falcon.crowdstrike.com/v1alpha1",
					Kind:       "FalconImageAnalyzer",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      ImageAnalyzerName,
					Namespace: testNamespace.Name,
				},
				Spec: falconv1alpha1.FalconImageAnalyzerSpec{
					InstallNamespace: imageAnalyzerNamespacedName.Namespace,
					FalconAPI: &falconv1alpha1.FalconAPI{
						ClientId:     "test-client-id",
						ClientSecret: "test-client-secret",
						CloudRegion:  "us-1",
					},
					Registry: falconv1alpha1.RegistrySpec{
						Type: "crowdstrike",
					},
					Image: imageAnalyzerImage,
					ImageAnalyzerConfig: falconv1alpha1.FalconImageAnalyzerConfigSpec{
						ImagePullSecrets: []corev1.LocalObjectReference{
							{Name: "extra-secret"},
						},
					},
				},
			}
			err := k8sClient.Create(ctx, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconImageAnalyzer{}
				return k8sClient.Get(ctx, imageAnalyzerNamespacedName, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Creating the reconciler")
			falconImageAnalyzerReconciler := &FalconImageAnalyzerReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			By("Creating initial service account with multiple imagePullSecrets")
			updated, err := falconImageAnalyzerReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: imageAnalyzerNamespacedName,
			}, log, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse())

			By("Verifying initial imagePullSecrets include extra secret")
			Eventually(func() int {
				found := &corev1.ServiceAccount{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.ImageServiceAccountName,
					Namespace: testNamespace.Name,
				}, found)
				return len(found.ImagePullSecrets)
			}, 5*time.Second, time.Second).Should(Equal(2))

			By("Removing extra imagePullSecret from FalconImageAnalyzer spec")
			falconImageAnalyzer.Spec.ImageAnalyzerConfig.ImagePullSecrets = []corev1.LocalObjectReference{}

			By("Calling reconcileServiceAccount with removed imagePullSecret")
			updated, err = falconImageAnalyzerReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: imageAnalyzerNamespacedName,
			}, log, falconImageAnalyzer)

			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeTrue(), "reconcileServiceAccount should return true when imagePullSecrets are removed")

			By("Verifying imagePullSecrets were reduced to default only")
			Eventually(func() []corev1.LocalObjectReference {
				found := &corev1.ServiceAccount{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.ImageServiceAccountName,
					Namespace: testNamespace.Name,
				}, found)
				return found.ImagePullSecrets
			}, 5*time.Second, time.Second).Should(Equal([]corev1.LocalObjectReference{
				{Name: common.FalconPullSecretName},
			}))
		})

		It("should not trigger reconciliation loop when external annotations are added", func() {
			log := zap.New(zap.UseDevMode(true))

			By("Creating the custom resource for the Kind FalconImageAnalyzer")
			falconImageAnalyzer := &falconv1alpha1.FalconImageAnalyzer{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "falcon.crowdstrike.com/v1alpha1",
					Kind:       "FalconImageAnalyzer",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      ImageAnalyzerName,
					Namespace: testNamespace.Name,
				},
				Spec: falconv1alpha1.FalconImageAnalyzerSpec{
					InstallNamespace: imageAnalyzerNamespacedName.Namespace,
					FalconAPI: &falconv1alpha1.FalconAPI{
						ClientId:     "test-client-id",
						ClientSecret: "test-client-secret",
						CloudRegion:  "us-1",
					},
					Registry: falconv1alpha1.RegistrySpec{
						Type: "crowdstrike",
					},
					Image: imageAnalyzerImage,
					ImageAnalyzerConfig: falconv1alpha1.FalconImageAnalyzerConfigSpec{
						ServiceAccount: falconv1alpha1.FalconImageAnalyzerServiceAccount{
							Annotations: map[string]string{
								"operator-managed-annotation": "value1",
							},
						},
					},
				},
			}
			err := k8sClient.Create(ctx, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconImageAnalyzer{}
				return k8sClient.Get(ctx, imageAnalyzerNamespacedName, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Creating the reconciler")
			falconImageAnalyzerReconciler := &FalconImageAnalyzerReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			By("Creating initial service account with operator-managed annotations")
			updated, err := falconImageAnalyzerReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: imageAnalyzerNamespacedName,
			}, log, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse())

			By("Verifying service account was created with operator-managed annotation")
			Eventually(func() map[string]string {
				found := &corev1.ServiceAccount{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.ImageServiceAccountName,
					Namespace: testNamespace.Name,
				}, found)
				return found.Annotations
			}, 5*time.Second, time.Second).Should(HaveKeyWithValue("operator-managed-annotation", "value1"))

			By("Simulating external system (like OpenShift) adding annotations to the service account")
			serviceAccount := &corev1.ServiceAccount{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      common.ImageServiceAccountName,
				Namespace: testNamespace.Name,
			}, serviceAccount)
			Expect(err).To(Not(HaveOccurred()))

			serviceAccount.Annotations["openshift.io/sa.scc.mcs"] = "s0:c27,c4"
			serviceAccount.Annotations["openshift.io/sa.scc.supplemental-groups"] = "1000710000/10000"
			serviceAccount.Annotations["openshift.io/sa.scc.uid-range"] = "1000710000/10000"
			err = k8sClient.Update(ctx, serviceAccount)
			Expect(err).To(Not(HaveOccurred()))

			By("Verifying external annotations were added")
			Eventually(func() bool {
				found := &corev1.ServiceAccount{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.ImageServiceAccountName,
					Namespace: testNamespace.Name,
				}, found)
				return len(found.Annotations) == 4 // 1 operator-managed + 3 openshift
			}, 5*time.Second, time.Second).Should(BeTrue())

			By("Calling reconcileServiceAccount again - should not trigger update for external annotations")
			updated, err = falconImageAnalyzerReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: imageAnalyzerNamespacedName,
			}, log, falconImageAnalyzer)

			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse(), "reconcileServiceAccount should return false when only external annotations exist")

			By("Verifying external annotations are preserved after reconciliation")
			serviceAccount = &corev1.ServiceAccount{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      common.ImageServiceAccountName,
				Namespace: testNamespace.Name,
			}, serviceAccount)
			Expect(err).To(Not(HaveOccurred()))
			Expect(serviceAccount.Annotations).To(HaveKeyWithValue("openshift.io/sa.scc.mcs", "s0:c27,c4"))
			Expect(serviceAccount.Annotations).To(HaveKeyWithValue("openshift.io/sa.scc.supplemental-groups", "1000710000/10000"))
			Expect(serviceAccount.Annotations).To(HaveKeyWithValue("openshift.io/sa.scc.uid-range", "1000710000/10000"))
			Expect(serviceAccount.Annotations).To(HaveKeyWithValue("operator-managed-annotation", "value1"))
		})
	})
})
