package controllers

import (
	"context"
	"fmt"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("FalconAdmission controller", func() {
	Context("FalconAdmission controller test", func() {

		const AdmissionControllerName = "test-falconadmissioncontroller"
		const AdmissionControllerNamespace = "falcon-kac"
		admissionImage := "example.com/image:test"
		falconCID := "1234567890ABCDEF1234567890ABCDEF-12"

		ctx := context.Background()

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      AdmissionControllerNamespace,
				Namespace: AdmissionControllerNamespace,
			},
		}

		typeNamespaceName := types.NamespacedName{Name: AdmissionControllerName, Namespace: AdmissionControllerNamespace}

		BeforeEach(func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, namespace)
			Expect(err).To(Not(HaveOccurred()))
		})

		AfterEach(func() {
			// TODO(user): Attention if you improve this code by adding other context test you MUST
			// be aware of the current delete namespace limitations. More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Deleting the Namespace to perform the tests")
			_ = k8sClient.Delete(ctx, namespace)
		})

		It("should successfully reconcile a custom resource for FalconAdmission", func() {
			By("Creating the custom resource for the Kind FalconAdmission")
			falconAdmission := &falconv1alpha1.FalconAdmission{}
			err := k8sClient.Get(ctx, typeNamespaceName, falconAdmission)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				falconAdmission := &falconv1alpha1.FalconAdmission{
					ObjectMeta: metav1.ObjectMeta{
						Name:      AdmissionControllerName,
						Namespace: AdmissionControllerNamespace,
					},
					Spec: falconv1alpha1.FalconAdmissionSpec{
						Falcon: falconv1alpha1.FalconSensor{
							CID: &falconCID,
						},
						InstallNamespace: "falcon-kac",
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

				err = k8sClient.Create(ctx, falconAdmission)
				Expect(err).To(Not(HaveOccurred()))
			}

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconAdmission{}
				return k8sClient.Get(ctx, typeNamespaceName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			falconAdmissionReconciler := &FalconAdmissionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = falconAdmissionReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if Service Account was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.ServiceAccount{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: "falcon-operator-admission-controller", Namespace: AdmissionControllerNamespace}, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking if ResourceQuota was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.ResourceQuota{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: "test-falconadmissioncontroller", Namespace: AdmissionControllerNamespace}, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking if TLS Secret was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.Secret{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: "test-falconadmissioncontroller-tls", Namespace: AdmissionControllerNamespace}, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking if ConfigMap was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.ConfigMap{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: "test-falconadmissioncontroller-config", Namespace: AdmissionControllerNamespace}, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking if Service was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.Service{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: "test-falconadmissioncontroller", Namespace: AdmissionControllerNamespace}, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func() error {
				found := &appsv1.Deployment{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: "test-falconadmissioncontroller", Namespace: AdmissionControllerNamespace}, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking if pods were successfully created in the reconciliation")
			Eventually(func() error {
				pod, err := k8sutils.GetReadyPod(k8sClient, ctx, AdmissionControllerNamespace, map[string]string{common.FalconComponentKey: common.FalconAdmissionController})
				if err != nil && err != k8sutils.ErrNoWebhookServicePodReady {
					return err
				}
				if pod.Name == "" {
					_, err = falconAdmissionReconciler.Reconcile(ctx, reconcile.Request{
						NamespacedName: typeNamespaceName,
					})
				}
				return err
			}, time.Minute, time.Second).Should(Succeed())

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
			}, time.Minute, time.Second).Should(Succeed())
		})
	})
})
