package falcon

import (
	"context"
	"fmt"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
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
			By("Cleaning up previously used Namespace and shared resources")

			// Delete all deployments
			deployList := &appsv1.DeploymentList{}
			Expect(k8sClient.List(ctx, deployList, client.InNamespace(sensorNamespacedName.Namespace))).To(Succeed())
			for _, item := range deployList.Items {
				Expect(k8sClient.Delete(ctx, &item)).To(Succeed())
			}

			Eventually(func() int {
				deployList := &appsv1.DeploymentList{}
				_ = k8sClient.List(ctx, deployList, client.InNamespace(sensorNamespacedName.Namespace))
				return len(deployList.Items)
			}, 6*time.Second, 2*time.Second).Should(Equal(0))

			// Delete cluster level resources
			clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: common.NodeClusterRoleBindingName}, clusterRoleBinding)).To(Succeed())
			Expect(k8sClient.Delete(ctx, clusterRoleBinding)).To(Succeed())

			// Delete FalconNodeSensor custom resource
			falconNodeSensorCR := &falconv1alpha1.FalconNodeSensor{}
			Expect(k8sClient.Get(ctx, sensorNamespacedName, falconNodeSensorCR)).To(Succeed())

			// Remove finalizer for successful FalconNodeSensor CR deletion
			patch := client.MergeFrom(falconNodeSensorCR.DeepCopy())
			falconNodeSensorCR.SetFinalizers(nil)
			_ = k8sClient.Patch(ctx, falconNodeSensorCR, patch)

			Expect(k8sClient.Delete(ctx, falconNodeSensorCR)).To(Succeed())

			Eventually(func() bool {
				falconNodeSensorCR := &falconv1alpha1.FalconNodeSensor{}
				err := k8sClient.Get(ctx, sensorNamespacedName, falconNodeSensorCR)
				return errors.IsNotFound(err)
			}, 6*time.Second, 2*time.Second).Should(BeTrue())

			_ = k8sClient.Delete(ctx, testNamespace)
		})

		It("should successfully reconcile a custom resource for FalconNodeSensor", func() {
			By("Creating the custom resource for the Kind FalconNodeSensor")
			falconNode := &falconv1alpha1.FalconNodeSensor{}
			err := k8sClient.Get(ctx, sensorNamespacedName, falconNode)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				falconNode := &falconv1alpha1.FalconNodeSensor{
					ObjectMeta: metav1.ObjectMeta{
						Name:      NodeSensorName,
						Namespace: sensorNamespacedName.Namespace,
					},
					Spec: falconv1alpha1.FalconNodeSensorSpec{
						Falcon: falconv1alpha1.FalconSensor{
							CID: &falconCID,
						},
						Node: falconv1alpha1.FalconNodeSensorConfig{
							Image: "example.com/image:test",
						},
						InstallNamespace: sensorNamespacedName.Namespace,
					},
				}

				err = k8sClient.Create(ctx, falconNode)
				Expect(err).To(Not(HaveOccurred()))
			}

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconNodeSensor{}
				return k8sClient.Get(ctx, sensorNamespacedName, found)
			}, 10*time.Second, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			tracker, cancel := sensorversion.NewTestTracker()
			defer cancel()

			falconNodeReconciler := &FalconNodeSensorReconciler{
				Client:  k8sClient,
				Reader:  k8sReader,
				Scheme:  k8sClient.Scheme(),
				tracker: tracker,
			}

			_, err = falconNodeReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: sensorNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			_, err = falconNodeReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: sensorNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			// TODO: serviceAccount reconciliation might be removed in the future
			_, err = falconNodeReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: sensorNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			//// TODO: clusterRoleBinding reconciliation might be removed in the future
			_, err = falconNodeReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: sensorNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if ConfigMap was successfully created in the reconciliation")
			Eventually(func() error {
				found := &corev1.ConfigMap{}
				cmName := NodeSensorName + "-config"
				return k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: sensorNamespacedName.Namespace}, found)
			}, 10*time.Second, time.Second).Should(Succeed())

			_, err = falconNodeReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: sensorNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if DaemonSet was successfully created in the reconciliation")
			Eventually(func() error {
				found := &appsv1.DaemonSet{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: NodeSensorName, Namespace: sensorNamespacedName.Namespace}, found)
			}, 10*time.Second, time.Second).Should(Succeed())

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
			}, 10*time.Second, time.Second).Should(Succeed())
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

			By("Creating the FalconNodeSensor CR with FalconSecret configured")
			falconNode := &falconv1alpha1.FalconNodeSensor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      NodeSensorName,
					Namespace: sensorNamespacedName.Namespace,
				},
				Spec: falconv1alpha1.FalconNodeSensorSpec{
					Falcon: falconv1alpha1.FalconSensor{
						CID: &falconCID,
					},
					Node: falconv1alpha1.FalconNodeSensorConfig{
						Image: "example.com/image:test",
					},
					FalconSecret: falconv1alpha1.FalconSecret{
						Enabled:    true,
						Namespace:  testSecretNamespace,
						SecretName: secretName,
					},
					InstallNamespace: sensorNamespacedName.Namespace,
				},
			}

			err = k8sClient.Create(ctx, falconNode)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				falconNode := &falconv1alpha1.FalconNodeSensor{}
				return k8sClient.Get(ctx, sensorNamespacedName, falconNode)
			}, 6*time.Second, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			tracker, cancel := sensorversion.NewTestTracker()
			defer cancel()

			falconNodeSensorReconciler := &FalconNodeSensorReconciler{
				Client:  k8sClient,
				Reader:  k8sReader,
				Scheme:  k8sClient.Scheme(),
				tracker: tracker,
			}

			// FalconNodeSensor needs to reconcile 5 times to complete all steps of the reconciler
			for range 5 {
				_, err = falconNodeSensorReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: sensorNamespacedName,
				})
				Expect(err).To(Not(HaveOccurred()))
			}

			By("Checking if Deployment was created with updated config map")
			Eventually(func() error {
				nodeSensorConfigMap := &corev1.ConfigMap{}
				err = common.GetNamespacedObject(
					ctx,
					falconNodeSensorReconciler.Client,
					falconNodeSensorReconciler.Reader,
					types.NamespacedName{Name: NodeSensorName + "-config", Namespace: sensorNamespacedName.Namespace},
					nodeSensorConfigMap,
				)
				if err != nil {
					return fmt.Errorf("failed to get nodeSensor configmap: %w", err)
				}

				nodeSensorDaemonSet := &appsv1.DaemonSet{}
				err = k8sClient.Get(ctx, types.NamespacedName{Name: NodeSensorName, Namespace: sensorNamespacedName.Namespace}, nodeSensorDaemonSet)
				if err != nil {
					return fmt.Errorf("failed to get nodeSensor daemonset: %w", err)
				}

				// Check for environment variables with secret references
				containers := nodeSensorDaemonSet.Spec.Template.Spec.Containers
				if len(containers) == 0 {
					return fmt.Errorf("no containers found in nodeSensor deployment")
				}

				expectedNodeSensorContainerName := "falcon-node-sensor"
				Expect(nodeSensorDaemonSet).To(haveContainerNamed(expectedNodeSensorContainerName))
				Expect(nodeSensorDaemonSet).To(haveContainerWithConfigMapEnvFrom(expectedNodeSensorContainerName, NodeSensorName+"-config"))
				Expect(nodeSensorConfigMap.Data["FALCONCTL_OPT_CID"]).To(Equal(falconCID))
				Expect(nodeSensorConfigMap.Data["FALCONCTL_OPT_PROVISIONING_TOKEN"]).To(Equal(provisioningToken))

				return nil
			}, 10*time.Second, time.Second).Should(Succeed())

			By("Cleaning up the test specific resources")
			err = k8sClient.Delete(ctx, testSecret)
			Expect(err).To(Not(HaveOccurred()))
		})
	})
})
