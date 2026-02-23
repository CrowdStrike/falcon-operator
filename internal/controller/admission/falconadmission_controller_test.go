package controllers

import (
	"context"
	"fmt"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	internalErrors "github.com/crowdstrike/falcon-operator/internal/errors"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("FalconAdmission controller", func() {
	Context("FalconAdmission controller test", func() {

		const (
			AdmissionControllerName      = "test-falconadmissioncontroller"
			AdmissionControllerNamespace = "falcon-kac"
		)

		var (
			// The namespaceCounter is a way to create a unique namespace for each test. Namespaces in tests are not reusable.
			// ref: https://book.kubebuilder.io/reference/envtest.html#namespace-usage-limitation
			namespaceCounter                int
			namespace                       *corev1.Namespace
			namespaceName                   string
			falconAdmission                 *falconv1alpha1.FalconAdmission
			controllerName                  string
			admissionNamespacedName         types.NamespacedName
			shouldAdmissionControlBeEnabled bool
			resourceQuotaName               string
			tlsSecretName                   string
			configMapName                   string
			serviceName                     string
			deploymentName                  string
		)

		admissionImage := "example.com/image:test"
		falconCID := "1234567890ABCDEF1234567890ABCDEF-12"
		ctx := context.Background()

		BeforeEach(func() {
			namespaceCounter++
			namespaceName = fmt.Sprintf("%s-%d", AdmissionControllerNamespace, namespaceCounter)
			controllerName = fmt.Sprintf("%s-%d", AdmissionControllerName, namespaceCounter)
			admissionNamespacedName = types.NamespacedName{Name: controllerName, Namespace: namespaceName}
			resourceQuotaName = controllerName
			tlsSecretName = fmt.Sprintf("%s-tls", controllerName)
			configMapName = fmt.Sprintf("%s-config", controllerName)
			serviceName = controllerName
			deploymentName = controllerName

			By(fmt.Sprintf("Creating the Namespace %s to perform the tests", namespaceName))
			namespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespaceName,
					Namespace: namespaceName,
				},
			}
			err := k8sClient.Create(ctx, namespace)
			Expect(err).To(Not(HaveOccurred()))

			falconAdmission = &falconv1alpha1.FalconAdmission{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "falcon.crowdstrike.com/v1alpha1",
					Kind:       "FalconAdmission",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      controllerName,
					Namespace: namespaceName,
				},
				Spec: falconv1alpha1.FalconAdmissionSpec{
					Falcon: falconv1alpha1.FalconSensor{
						CID: &falconCID,
					},
					InstallNamespace: namespaceName,
					Image:            admissionImage,
					Registry: falconv1alpha1.RegistrySpec{
						Type: "crowdstrike",
					},
					AdmissionConfig: falconv1alpha1.FalconAdmissionConfigSpec{
						DepUpdateStrategy: falconv1alpha1.FalconAdmissionUpdateStrategy{
							RollingUpdate: appsv1.RollingUpdateDeployment{
								MaxUnavailable: &intstr.IntOrString{IntVal: 1},
								MaxSurge:       &intstr.IntOrString{IntVal: 1},
							},
						},
					},
				},
			}
		})

		AfterEach(func() {
			// be aware of the current delete namespace limitations. More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Cleaning up previously used Namespace and shared resources")

			// Delete all deployments
			deployList := &appsv1.DeploymentList{}
			Expect(k8sClient.List(ctx, deployList, client.InNamespace(namespaceName))).To(Succeed())
			for _, item := range deployList.Items {
				Expect(k8sClient.Delete(ctx, &item)).To(Succeed())
			}

			Eventually(func() int {
				deployList := &appsv1.DeploymentList{}
				_ = k8sClient.List(ctx, deployList, client.InNamespace(namespaceName))
				return len(deployList.Items)
			}, 6*time.Second, 2*time.Second).Should(Equal(0))

			// Delete FalconAdmission custom resource
			falconAdmissionCR := &falconv1alpha1.FalconAdmission{}
			Expect(k8sClient.Get(ctx, admissionNamespacedName, falconAdmissionCR)).To(Succeed())
			Expect(k8sClient.Delete(ctx, falconAdmissionCR)).To(Succeed())

			// Delete cluster level resources
			clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: admissionClusterRoleBindingName}, clusterRoleBinding); err == nil {
				Expect(k8sClient.Delete(ctx, clusterRoleBinding)).To(Succeed())
			}

			validatingWebhookConfig := &arv1.ValidatingWebhookConfiguration{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: common.FalconAdmissionValidatingWebhookName}, validatingWebhookConfig)
			if err == nil {
				Expect(k8sClient.Delete(ctx, validatingWebhookConfig)).To(Succeed())
			}

			_ = k8sClient.Delete(ctx, namespace)
		})

		// Validating a generic deployment of FalconAdmission
		It("should successfully reconcile a custom resource for FalconAdmission", func() {
			shouldAdmissionControlBeEnabled = true

			By("Creating the custom resource for the Kind FalconAdmission")
			err := k8sClient.Create(ctx, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconAdmission{}
				return k8sClient.Get(ctx, admissionNamespacedName, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			falconAdmissionReconciler := &FalconAdmissionReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}
			_, err = falconAdmissionReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if Service Account was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.ServiceAccount{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: common.AdmissionServiceAccountName, Namespace: namespaceName}, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Checking if ResourceQuota was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.ResourceQuota{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: resourceQuotaName, Namespace: namespaceName}, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Checking if TLS Secret was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.Secret{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: tlsSecretName, Namespace: namespaceName}, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Checking if ConfigMap was successfully created in the reconciliation")
			var configMap *corev1.ConfigMap
			Eventually(func() error {
				configMap = &corev1.ConfigMap{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespaceName}, configMap)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Checking if ConfigMap disabled Admission Control")
			Expect(configMap.Data["__CS_ADMISSION_CONTROL_ENABLED"]).To(Equal(fmt.Sprintf("%t", shouldAdmissionControlBeEnabled)))

			By("Checking if Service was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.Service{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespaceName}, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func() error {
				found := &appsv1.Deployment{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespaceName}, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Checking if pods were successfully created in the reconciliation")
			Eventually(func() error {
				pod, err := k8sutils.GetReadyPod(k8sClient, ctx, namespaceName, map[string]string{common.FalconComponentKey: common.FalconAdmissionController})
				if err != nil && err != k8sutils.ErrNoWebhookServicePodReady {
					return err
				}
				if pod.Name == "" {
					_, err = falconAdmissionReconciler.Reconcile(ctx, reconcile.Request{
						NamespacedName: admissionNamespacedName,
					})
				}
				return err
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Checking the latest Status Condition added to the FalconAdmission instance")
			Eventually(func() error {
				if len(falconAdmission.Status.Conditions) != 0 {
					latestStatusCondition := falconAdmission.Status.Conditions[len(falconAdmission.Status.Conditions)-1]
					expectedLatestStatusCondition := metav1.Condition{Type: falconv1alpha1.ConditionDeploymentReady,
						Status: metav1.ConditionTrue, Reason: falconv1alpha1.ReasonInstallSucceeded,
						Message: "FalconAdmission installation completed"}
					if latestStatusCondition != expectedLatestStatusCondition {
						return fmt.Errorf("The latest status condition added to the FalconAdmission instance is not as expected")
					}
				}
				return nil
			}, 20*time.Second, time.Second).Should(Succeed())
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
			falconAdmission.Spec.FalconSecret = falconv1alpha1.FalconSecret{
				Enabled:    true,
				Namespace:  testSecretNamespace,
				SecretName: secretName,
			}

			err = k8sClient.Create(ctx, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				falconAdmission := &falconv1alpha1.FalconAdmission{}
				return k8sClient.Get(ctx, admissionNamespacedName, falconAdmission)
			}, 6*time.Second, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			falconAdmissionReconciler := &FalconAdmissionReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			_, err = falconAdmissionReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if Service was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.Service{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespaceName}, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Checking if Deployment was created with updated config map")
			Eventually(func() error {
				admissionConfigMap := &corev1.ConfigMap{}
				err = common.GetNamespacedObject(
					ctx,
					falconAdmissionReconciler.Client,
					falconAdmissionReconciler.Reader,
					types.NamespacedName{Name: configMapName, Namespace: admissionNamespacedName.Namespace},
					admissionConfigMap,
				)
				if err != nil {
					return fmt.Errorf("failed to get admission configmap: %w", err)
				}

				admissionDeployment := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      deploymentName,
					Namespace: admissionNamespacedName.Namespace,
				}, admissionDeployment)
				if err != nil {
					return fmt.Errorf("failed to get admission deployment: %w", err)
				}

				// Check for environment variables with secret references
				containers := admissionDeployment.Spec.Template.Spec.Containers
				if len(containers) == 0 {
					return fmt.Errorf("no containers found in admission deployment")
				}

				Expect(admissionDeployment).To(haveContainerNamed("falcon-kac"))
				Expect(admissionDeployment).To(haveContainerWithConfigMapEnvFrom("falcon-kac", configMapName))
				Expect(admissionConfigMap.Data["FALCONCTL_OPT_CID"]).To(Equal(falconCID))
				Expect(admissionConfigMap.Data["FALCONCTL_OPT_PROVISIONING_TOKEN"]).To(Equal(provisioningToken))

				return nil
			}, 10*time.Second, time.Second).Should(Succeed())

			By("Checking if pods were successfully created in the reconciliation")
			Eventually(func() error {
				pod, err := k8sutils.GetReadyPod(k8sClient, ctx, namespaceName, map[string]string{common.FalconComponentKey: common.FalconAdmissionController})
				if err != nil && err != k8sutils.ErrNoWebhookServicePodReady {
					return err
				}
				if pod.Name == "" {
					_, err = falconAdmissionReconciler.Reconcile(ctx, reconcile.Request{
						NamespacedName: admissionNamespacedName,
					})
				}
				return err
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Checking the latest Status Condition added to the FalconAdmission instance")
			Eventually(func() error {
				if len(falconAdmission.Status.Conditions) != 0 {
					latestStatusCondition := falconAdmission.Status.Conditions[len(falconAdmission.Status.Conditions)-1]
					expectedLatestStatusCondition := metav1.Condition{Type: falconv1alpha1.ConditionDeploymentReady,
						Status: metav1.ConditionTrue, Reason: falconv1alpha1.ReasonInstallSucceeded,
						Message: "FalconAdmission installation completed"}
					if latestStatusCondition != expectedLatestStatusCondition {
						return fmt.Errorf("The latest status condition added to the FalconAdmission instance is not as expected")
					}
				}
				return nil
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Cleaning up the test specific resources")
			err = k8sClient.Delete(ctx, testSecret)
			Expect(err).To(Not(HaveOccurred()))
		})

		It("should error if falcon secret is enabled and Falcon API credentials are missing", func() {
			By("Creating test secrets - without Falcon API credentials")
			secretName := "falcon-secrets-1"
			testSecretNamespace := "falcon-secret-1"

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
					"falcon-cid": falconCID,
				},
			}
			err = k8sClient.Create(ctx, testSecret)
			Expect(err).To(Not(HaveOccurred()))

			By("Creating the FalconAdmission CR with FalconSecret configured")
			falconAdmission.Spec.FalconSecret = falconv1alpha1.FalconSecret{
				Enabled:    true,
				Namespace:  testSecretNamespace,
				SecretName: secretName,
			}
			falconAdmission.Spec.Image = ""

			err = k8sClient.Create(ctx, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				falconAdmission := &falconv1alpha1.FalconAdmission{}
				return k8sClient.Get(ctx, admissionNamespacedName, falconAdmission)
			}, 6*time.Second, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			falconAdmissionReconciler := &FalconAdmissionReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			_, err = falconAdmissionReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			})
			Expect(err).To(MatchError(ContainSubstring(internalErrors.ErrMissingFalconAPICredentialsInSecret.Error())))

			By("Cleaning up the test specific resources")
			err = k8sClient.Delete(ctx, testSecret)
			Expect(err).To(Not(HaveOccurred()))
		})

		// Testing when FalconAdmission disables the admission controller
		It("should successfully reconcile a custom resource for FalconAdmission - with admission control disabled", func() {
			shouldAdmissionControlBeEnabled = false
			falconAdmission.Spec.AdmissionConfig.AdmissionControlEnabled = &shouldAdmissionControlBeEnabled

			By("Creating the custom resource for the Kind FalconAdmission - with admission control disabled")
			err := k8sClient.Create(ctx, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created - with admission control disabled")
			Eventually(func() error {
				found := &falconv1alpha1.FalconAdmission{}
				return k8sClient.Get(ctx, admissionNamespacedName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Reconciling the custom resource create - with admission control disabled")
			falconAdmissionReconciler := &FalconAdmissionReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}
			_, err = falconAdmissionReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if Service was successfully created in the reconciliation - with admission control disabled")
			Eventually(func() error {
				found := &corev1.Service{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespaceName}, found)
			}, time.Minute, time.Second).ShouldNot(Succeed())

			By("Checking if ConfigMap was successfully created in the reconciliation - with admission control disabled")
			var configMap *corev1.ConfigMap
			Eventually(func() error {
				configMap = &corev1.ConfigMap{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespaceName}, configMap)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking if ConfigMap data exists - with admission control disabled")
			Expect(configMap.Data["__CS_ADMISSION_CONTROL_ENABLED"]).To(Equal(fmt.Sprintf("%t", shouldAdmissionControlBeEnabled)))

			By("Checking if Deployment was successfully created in the reconciliation - with admission control disabled")
			var deployment *appsv1.Deployment
			Eventually(func() error {
				deployment = &appsv1.Deployment{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespaceName}, deployment)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking if falcon-client container has default resources - with admission control disabled")
			var falconClientContainer *corev1.Container
			for _, container := range deployment.Spec.Template.Spec.Containers {
				if container.Name == "falcon-client" {
					falconClientContainer = &container
					break
				}
			}

			Expect(falconClientContainer).ToNot(BeNil())
			Expect(falconClientContainer.Resources.Requests.Cpu().String()).To(Equal("100m"))
			Expect(falconClientContainer.Resources.Requests.Memory().String()).To(Equal("128Mi"))
			Expect(falconClientContainer.Resources.Limits.Memory().String()).To(Equal("128Mi"))
		})

		// Testing reconcileServiceAccount return value
		It("should return false when creating a new service account", func() {
			log := zap.New(zap.UseDevMode(true))

			By("Creating the custom resource for the Kind FalconAdmission")
			err := k8sClient.Create(ctx, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconAdmission{}
				return k8sClient.Get(ctx, admissionNamespacedName, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Creating the reconciler")
			falconAdmissionReconciler := &FalconAdmissionReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			By("Calling reconcileServiceAccount when no service account exists")
			updated, err := falconAdmissionReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			}, log, falconAdmission)

			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse(), "reconcileServiceAccount should return false when creating a new service account")

			By("Verifying the service account was created")
			Eventually(func() error {
				found := &corev1.ServiceAccount{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.AdmissionServiceAccountName,
					Namespace: namespaceName,
				}, found)
			}, 5*time.Second, time.Second).Should(Succeed())
		})

		It("should return false when only annotations are updated", func() {
			log := zap.New(zap.UseDevMode(true))

			By("Creating the custom resource for the Kind FalconAdmission")
			err := k8sClient.Create(ctx, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconAdmission{}
				return k8sClient.Get(ctx, admissionNamespacedName, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Creating the reconciler")
			falconAdmissionReconciler := &FalconAdmissionReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			By("Creating initial service account")
			updated, err := falconAdmissionReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			}, log, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse())

			By("Updating annotations in FalconAdmission spec")
			falconAdmission.Spec.AdmissionConfig.ServiceAccount.Annotations = map[string]string{
				"new-annotation": "new-value",
			}

			By("Calling reconcileServiceAccount with updated annotations")
			updated, err = falconAdmissionReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			}, log, falconAdmission)

			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse(), "reconcileServiceAccount should return false when only annotations are updated")

			By("Verifying annotations were updated")
			Eventually(func() map[string]string {
				found := &corev1.ServiceAccount{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.AdmissionServiceAccountName,
					Namespace: namespaceName,
				}, found)
				return found.ObjectMeta.Annotations
			}, 5*time.Second, time.Second).Should(HaveKey("new-annotation"))
		})

		It("should return true when imagePullSecrets are updated", func() {
			log := zap.New(zap.UseDevMode(true))

			By("Creating the custom resource for the Kind FalconAdmission")
			err := k8sClient.Create(ctx, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconAdmission{}
				return k8sClient.Get(ctx, admissionNamespacedName, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Creating the reconciler")
			falconAdmissionReconciler := &FalconAdmissionReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			By("Creating initial service account with default imagePullSecrets")
			updated, err := falconAdmissionReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			}, log, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse())

			By("Verifying initial imagePullSecrets")
			Eventually(func() []corev1.LocalObjectReference {
				found := &corev1.ServiceAccount{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.AdmissionServiceAccountName,
					Namespace: namespaceName,
				}, found)
				return found.ImagePullSecrets
			}, 5*time.Second, time.Second).Should(Equal([]corev1.LocalObjectReference{
				{Name: common.FalconPullSecretName},
			}))

			By("Adding a new imagePullSecret to FalconAdmission spec")
			falconAdmission.Spec.AdmissionConfig.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: "additional-secret"},
			}

			By("Calling reconcileServiceAccount with updated imagePullSecrets")
			updated, err = falconAdmissionReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			}, log, falconAdmission)

			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeTrue(), "reconcileServiceAccount should return true when imagePullSecrets are updated")

			By("Verifying imagePullSecrets were updated")
			Eventually(func() []corev1.LocalObjectReference {
				found := &corev1.ServiceAccount{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.AdmissionServiceAccountName,
					Namespace: namespaceName,
				}, found)
				return found.ImagePullSecrets
			}, 5*time.Second, time.Second).Should(Equal([]corev1.LocalObjectReference{
				{Name: common.FalconPullSecretName},
				{Name: "additional-secret"},
			}))
		})

		It("should return false when no changes are made to service account", func() {
			log := zap.New(zap.UseDevMode(true))

			By("Creating the custom resource for the Kind FalconAdmission")
			err := k8sClient.Create(ctx, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconAdmission{}
				return k8sClient.Get(ctx, admissionNamespacedName, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Creating the reconciler")
			falconAdmissionReconciler := &FalconAdmissionReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			By("Creating initial service account")
			updated, err := falconAdmissionReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			}, log, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse())

			By("Calling reconcileServiceAccount again without any changes")
			updated, err = falconAdmissionReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			}, log, falconAdmission)

			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse(), "reconcileServiceAccount should return false when no changes are made")
		})

		It("should return true when imagePullSecrets are removed", func() {
			log := zap.New(zap.UseDevMode(true))

			By("Creating the custom resource for the Kind FalconAdmission with extra imagePullSecret")
			falconAdmission.Spec.AdmissionConfig.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: "extra-secret"},
			}
			err := k8sClient.Create(ctx, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconAdmission{}
				return k8sClient.Get(ctx, admissionNamespacedName, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Creating the reconciler")
			falconAdmissionReconciler := &FalconAdmissionReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			By("Creating initial service account with multiple imagePullSecrets")
			updated, err := falconAdmissionReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			}, log, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse())

			By("Verifying initial imagePullSecrets include extra secret")
			Eventually(func() int {
				found := &corev1.ServiceAccount{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.AdmissionServiceAccountName,
					Namespace: namespaceName,
				}, found)
				return len(found.ImagePullSecrets)
			}, 5*time.Second, time.Second).Should(Equal(2))

			By("Removing extra imagePullSecret from FalconAdmission spec")
			falconAdmission.Spec.AdmissionConfig.ImagePullSecrets = []corev1.LocalObjectReference{}

			By("Calling reconcileServiceAccount with removed imagePullSecret")
			updated, err = falconAdmissionReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			}, log, falconAdmission)

			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeTrue(), "reconcileServiceAccount should return true when imagePullSecrets are removed")

			By("Verifying imagePullSecrets were reduced to default only")
			Eventually(func() []corev1.LocalObjectReference {
				found := &corev1.ServiceAccount{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.AdmissionServiceAccountName,
					Namespace: namespaceName,
				}, found)
				return found.ImagePullSecrets
			}, 5*time.Second, time.Second).Should(Equal([]corev1.LocalObjectReference{
				{Name: common.FalconPullSecretName},
			}))
		})

		It("should not trigger reconciliation loop when external annotations are added", func() {
			log := zap.New(zap.UseDevMode(true))

			By("Creating the custom resource for the Kind FalconAdmission with operator-managed annotations")
			falconAdmission.Spec.AdmissionConfig.ServiceAccount.Annotations = map[string]string{
				"operator-managed-annotation": "value1",
			}
			err := k8sClient.Create(ctx, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconAdmission{}
				return k8sClient.Get(ctx, admissionNamespacedName, found)
			}, 20*time.Second, time.Second).Should(Succeed())

			By("Creating the reconciler")
			falconAdmissionReconciler := &FalconAdmissionReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}

			By("Creating initial service account with operator-managed annotations")
			updated, err := falconAdmissionReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			}, log, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse())

			By("Verifying service account was created with operator-managed annotation")
			Eventually(func() map[string]string {
				found := &corev1.ServiceAccount{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.AdmissionServiceAccountName,
					Namespace: namespaceName,
				}, found)
				return found.Annotations
			}, 5*time.Second, time.Second).Should(HaveKeyWithValue("operator-managed-annotation", "value1"))

			By("Simulating external system (like OpenShift) adding annotations to the service account")
			serviceAccount := &corev1.ServiceAccount{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      common.AdmissionServiceAccountName,
				Namespace: namespaceName,
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
					Name:      common.AdmissionServiceAccountName,
					Namespace: namespaceName,
				}, found)
				return len(found.Annotations) == 4 // 1 operator-managed + 3 openshift
			}, 5*time.Second, time.Second).Should(BeTrue())

			By("Calling reconcileServiceAccount again - should not trigger update for external annotations")
			updated, err = falconAdmissionReconciler.reconcileServiceAccount(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			}, log, falconAdmission)

			Expect(err).To(Not(HaveOccurred()))
			Expect(updated).To(BeFalse(), "reconcileServiceAccount should return false when only external annotations exist")

			By("Verifying external annotations are preserved after reconciliation")
			serviceAccount = &corev1.ServiceAccount{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      common.AdmissionServiceAccountName,
				Namespace: namespaceName,
			}, serviceAccount)
			Expect(err).To(Not(HaveOccurred()))
			Expect(serviceAccount.Annotations).To(HaveKeyWithValue("openshift.io/sa.scc.mcs", "s0:c27,c4"))
			Expect(serviceAccount.Annotations).To(HaveKeyWithValue("openshift.io/sa.scc.supplemental-groups", "1000710000/10000"))
			Expect(serviceAccount.Annotations).To(HaveKeyWithValue("openshift.io/sa.scc.uid-range", "1000710000/10000"))
			Expect(serviceAccount.Annotations).To(HaveKeyWithValue("operator-managed-annotation", "value1"))
		})
	})
})
