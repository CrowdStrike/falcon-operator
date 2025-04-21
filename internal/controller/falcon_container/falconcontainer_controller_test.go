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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("FalconContainer controller", func() {
	Context("FalconContainer controller test", func() {

		const SidecarSensorName = "test-falconsidecarsensor"
		const SidecarSensorNamespace = "falcon-system"
		containerImage := "example.com/image:test"
		falconCID := "1234567890ABCDEF1234567890ABCDEF-12"

		ctx := context.Background()

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SidecarSensorName,
				Namespace: SidecarSensorNamespace,
			},
		}

		typeNamespaceName := types.NamespacedName{Name: SidecarSensorName, Namespace: SidecarSensorNamespace}

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

		It("should successfully reconcile a custom resource for FalconContainer", func() {
			By("Creating the custom resource for the Kind FalconContainer")
			falconContainer := &falconv1alpha1.FalconContainer{}
			err := k8sClient.Get(ctx, typeNamespaceName, falconContainer)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				falconContainer := &falconv1alpha1.FalconContainer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      SidecarSensorName,
						Namespace: namespace.Name,
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
				return k8sClient.Get(ctx, typeNamespaceName, found)
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
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			_, err = falconContainerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			// TODO: serviceAccount reconciliation might be removed in the future
			_, err = falconContainerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			// TODO: clusterRoleBinding reconciliation might be removed in the future
			_, err = falconContainerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
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
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func() error {
				found := &appsv1.Deployment{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: "falcon-sidecar-injector", Namespace: SidecarSensorNamespace}, found)
			}, time.Minute, time.Second).Should(Succeed())

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
						NamespacedName: typeNamespaceName,
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
	})
})
