package falcon

import (
	"context"
	"fmt"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/common/sensorversion"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("FalconNodeSensor controller", func() {
	Context("FalconNodeSensor controller test", func() {

		const NodeSensorName = "test-falconnodesensor"
		const NodeSensorNamespace = "falcon-system"
		falconCID := "1234567890ABCDEF1234567890ABCDEF-12"

		ctx := context.Background()

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      NodeSensorName,
				Namespace: NodeSensorName,
			},
		}

		typeNamespaceName := types.NamespacedName{Name: NodeSensorName, Namespace: NodeSensorName}

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

		It("should successfully reconcile a custom resource for FalconNodeSensor", func() {
			By("Creating the custom resource for the Kind FalconNodeSensor")
			falconNode := &falconv1alpha1.FalconNodeSensor{}
			err := k8sClient.Get(ctx, typeNamespaceName, falconNode)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				falconNode := &falconv1alpha1.FalconNodeSensor{
					ObjectMeta: metav1.ObjectMeta{
						Name:      NodeSensorName,
						Namespace: namespace.Name,
					},
					Spec: falconv1alpha1.FalconNodeSensorSpec{
						Falcon: falconv1alpha1.FalconSensor{
							CID: &falconCID,
						},
						Node: falconv1alpha1.FalconNodeSensorConfig{
							Image: "example.com/image:test",
						},
					},
				}

				err = k8sClient.Create(ctx, falconNode)
				Expect(err).To(Not(HaveOccurred()))
			}

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconNodeSensor{}
				return k8sClient.Get(ctx, typeNamespaceName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			tracker, cancel := sensorversion.NewTestTracker()
			defer cancel()

			falconNodeReconciler := &FalconNodeSensorReconciler{
				Client:  k8sClient,
				Scheme:  k8sClient.Scheme(),
				tracker: tracker,
			}

			_, err = falconNodeReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			_, err = falconNodeReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			// TODO: serviceAccount reconciliation might be removed in the future
			_, err = falconNodeReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			// TODO: clusterRoleBinding reconciliation might be removed in the future
			_, err = falconNodeReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if ConfigMap was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.ConfigMap{}
				cmName := NodeSensorName + "-config"
				return k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: NodeSensorNamespace}, found)
			}, time.Minute, time.Second).Should(Succeed())

			_, err = falconNodeReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if DaemonSet was successfully created in the reconciliation")
			Eventually(func() error {
				found := &appsv1.DaemonSet{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: NodeSensorName, Namespace: NodeSensorNamespace}, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking the latest Status Condition added to the FalconNodeSensor instance")
			Eventually(func() error {
				if len(falconNode.Status.Conditions) != 0 {
					latestStatusCondition := falconNode.Status.Conditions[len(falconNode.Status.Conditions)-1]
					expectedLatestStatusCondition := metav1.Condition{Type: falconv1alpha1.ConditionDaemonSetReady,
						Status: metav1.ConditionTrue, Reason: falconv1alpha1.ReasonInstallSucceeded,
						Message: "FalconNodeSensor DaemonSet has been successfully installed"}
					if latestStatusCondition != expectedLatestStatusCondition {
						return fmt.Errorf("The latest status condition added to the FalconNodeSensor instance is not as expected")
					}
				}
				return nil
			}, time.Minute, time.Second).Should(Succeed())
		})
	})
})
