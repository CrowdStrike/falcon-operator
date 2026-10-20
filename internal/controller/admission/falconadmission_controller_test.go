package controllers

import (
	"context"
	"fmt"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
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
			namespaceCounter        int
			namespace               *corev1.Namespace
			namespaceName           string
			falconAdmission         *falconv1alpha1.FalconAdmission
			controllerName          string
			admissionNamespacedName types.NamespacedName
		)

		admissionImage := "example.com/image:test"
		falconCID := "1234567890ABCDEF1234567890ABCDEF-12"
		ctx := context.Background()

		BeforeEach(func() {
			namespaceCounter++
			namespaceName = fmt.Sprintf("%s-%d", AdmissionControllerNamespace, namespaceCounter)
			controllerName = fmt.Sprintf("%s-%d", AdmissionControllerName, namespaceCounter)
			admissionNamespacedName = types.NamespacedName{Name: controllerName, Namespace: namespaceName}

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
							RollingUpdate: falconv1alpha1.FalconAdmissionRollingUpdate{
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

		It("should set the Deprecated condition and not deploy resources", func() {
			By("Creating the custom resource for the Kind FalconAdmission")
			Expect(k8sClient.Create(ctx, falconAdmission)).To(Succeed())

			By("Reconciling the custom resource created")
			falconAdmissionReconciler := &FalconAdmissionReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}
			_, err := falconAdmissionReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: admissionNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking the Deprecated status condition")
			found := &falconv1alpha1.FalconAdmission{}
			Expect(k8sClient.Get(ctx, admissionNamespacedName, found)).To(Succeed())
			cond := meta.FindStatusCondition(found.Status.Conditions, falconv1alpha1.ConditionDeprecated)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			Expect(cond.Reason).To(Equal(falconv1alpha1.ReasonDeprecated))

			By("Checking that no Deployment was created")
			deployList := &appsv1.DeploymentList{}
			Expect(k8sClient.List(ctx, deployList, client.InNamespace(namespaceName))).To(Succeed())
			Expect(deployList.Items).To(BeEmpty())
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
