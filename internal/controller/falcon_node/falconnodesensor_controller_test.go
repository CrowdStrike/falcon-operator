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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("FalconNodeSensor controller", func() {
	Context("FalconNodeSensor controller test", func() {

		const NodeSensorName = "test-falconnodesensor"
		const NodeSensorNamespace = "falcon-system"
		// The namespaceCounter is a way to create a unique testNamespace for each test. Namespaces in tests are not reusable.
		// ref: https://book.kubebuilder.io/reference/envtest.html#namespace-usage-limitation
		namespaceCounter := 0
		var testNamespace *corev1.Namespace
		var sensorNamespacedName types.NamespacedName

		falconCID := "1234567890ABCDEF1234567890ABCDEF-12"
		ctx := context.Background()

		BeforeEach(func() {
			namespaceCounter += 1
			currentNamespaceString := fmt.Sprintf("%s-%d", NodeSensorNamespace, namespaceCounter)
			testNamespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      currentNamespaceString,
					Namespace: currentNamespaceString,
				},
			}

			sensorNamespacedName = types.NamespacedName{Name: NodeSensorName, Namespace: currentNamespaceString}

			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, testNamespace)
			Expect(err).To(Not(HaveOccurred()))
		})

		AfterEach(func() {
			// be aware of the current delete namespace limitations. More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Cleaning up previously used Namespace")

			falconNodeSensorCR := &falconv1alpha1.FalconNodeSensor{}
			if err := k8sClient.Get(ctx, sensorNamespacedName, falconNodeSensorCR); err == nil {
				patch := client.MergeFrom(falconNodeSensorCR.DeepCopy())
				falconNodeSensorCR.SetFinalizers(nil)
				_ = k8sClient.Patch(ctx, falconNodeSensorCR, patch)
				_ = k8sClient.Delete(ctx, falconNodeSensorCR)
			}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, sensorNamespacedName, &falconv1alpha1.FalconNodeSensor{})
				return errors.IsNotFound(err)
			}, 6*time.Second, 2*time.Second).Should(BeTrue())

			_ = k8sClient.Delete(ctx, testNamespace)
		})

		newNodeSensor := func(finalizers ...string) *falconv1alpha1.FalconNodeSensor {
			return &falconv1alpha1.FalconNodeSensor{
				ObjectMeta: metav1.ObjectMeta{
					Name:       NodeSensorName,
					Namespace:  sensorNamespacedName.Namespace,
					Finalizers: finalizers,
				},
				Spec: falconv1alpha1.FalconNodeSensorSpec{
					Falcon: falconv1alpha1.FalconUnified{
						FalconSensor: falconv1alpha1.FalconSensor{
							CID: &falconCID,
						},
					},
					Node: falconv1alpha1.FalconNodeSensorConfig{
						Image: "example.com/image:test",
					},
					InstallNamespace: sensorNamespacedName.Namespace,
				},
			}
		}

		newReconciler := func() *FalconNodeSensorReconciler {
			return &FalconNodeSensorReconciler{
				Client: k8sClient,
				Reader: k8sReader,
				Scheme: k8sClient.Scheme(),
			}
		}

		It("should set the Deprecated condition and not deploy resources", func() {
			By("Creating the custom resource for the Kind FalconNodeSensor")
			Expect(k8sClient.Create(ctx, newNodeSensor())).To(Succeed())

			By("Reconciling the custom resource created")
			_, err := newReconciler().Reconcile(ctx, reconcile.Request{NamespacedName: sensorNamespacedName})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking the Deprecated status condition")
			found := &falconv1alpha1.FalconNodeSensor{}
			Expect(k8sClient.Get(ctx, sensorNamespacedName, found)).To(Succeed())
			cond := meta.FindStatusCondition(found.Status.Conditions, falconv1alpha1.ConditionDeprecated)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			Expect(cond.Reason).To(Equal(falconv1alpha1.ReasonDeprecated))

			By("Checking that no DaemonSet was created")
			dsList := &appsv1.DaemonSetList{}
			Expect(k8sClient.List(ctx, dsList, client.InNamespace(sensorNamespacedName.Namespace))).To(Succeed())
			Expect(dsList.Items).To(BeEmpty())
		})

		It("should remove a finalizer left by a previous operator version on deletion", func() {
			By("Creating a FalconNodeSensor that still carries the legacy finalizer")
			Expect(k8sClient.Create(ctx, newNodeSensor(common.FalconFinalizer))).To(Succeed())

			By("Deleting the custom resource")
			Expect(k8sClient.Delete(ctx, newNodeSensor())).To(Succeed())

			By("Reconciling the deleted custom resource")
			_, err := newReconciler().Reconcile(ctx, reconcile.Request{NamespacedName: sensorNamespacedName})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking that the finalizer was removed and the custom resource is gone")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, sensorNamespacedName, &falconv1alpha1.FalconNodeSensor{})
				return errors.IsNotFound(err)
			}, 6*time.Second, time.Second).Should(BeTrue())
		})
	})
})
