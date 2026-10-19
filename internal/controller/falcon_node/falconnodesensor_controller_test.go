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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

		XIt("should successfully reconcile a custom resource for FalconNodeSensor", func() {
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

			By("Checking that the finalizer was added (deprecated controller only adds finalizer)")
			Eventually(func() bool {
				found := &falconv1alpha1.FalconNodeSensor{}
				if err := k8sClient.Get(ctx, sensorNamespacedName, found); err != nil {
					return false
				}
				return controllerutil.ContainsFinalizer(found, common.FalconFinalizer)
			}, 10*time.Second, time.Second).Should(BeTrue())

			By("Checking that no ConfigMap is created (deprecated controller does not create resources)")
			Consistently(func() bool {
				found := &corev1.ConfigMap{}
				cmName := NodeSensorName + "-config"
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: sensorNamespacedName.Namespace}, found)
				return errors.IsNotFound(err)
			}, 3*time.Second, time.Second).Should(BeTrue())
		})

		XIt("should correctly handle and inject existing secrets into configmap", func() {
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
			cloudRegion := "us-1"
			falconNode := &falconv1alpha1.FalconNodeSensor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      NodeSensorName,
					Namespace: sensorNamespacedName.Namespace,
				},
				Spec: falconv1alpha1.FalconNodeSensorSpec{
					Falcon: falconv1alpha1.FalconUnified{
						FalconSensor: falconv1alpha1.FalconSensor{
							CID: &falconCID,
						},
						Cloud: cloudRegion,
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

			// FalconNodeSensor is deprecated. Reconcile only adds a finalizer.
			_, err = falconNodeSensorReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: sensorNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking that the finalizer was added (deprecated controller only adds finalizer)")
			Eventually(func() bool {
				found := &falconv1alpha1.FalconNodeSensor{}
				if err := k8sClient.Get(ctx, sensorNamespacedName, found); err != nil {
					return false
				}
				return controllerutil.ContainsFinalizer(found, common.FalconFinalizer)
			}, 10*time.Second, time.Second).Should(BeTrue())

			By("Checking that no ConfigMap is created (deprecated controller does not create resources)")
			Consistently(func() bool {
				found := &corev1.ConfigMap{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: NodeSensorName + "-config", Namespace: sensorNamespacedName.Namespace}, found)
				return errors.IsNotFound(err)
			}, 3*time.Second, time.Second).Should(BeTrue())

			By("Cleaning up the test specific resources")
			err = k8sClient.Delete(ctx, testSecret)
			Expect(err).To(Not(HaveOccurred()))
		})

		XIt("should add annotations to service account when configured", func() {
			By("Creating the FalconNodeSensor CR with service account annotations")
			falconNode := &falconv1alpha1.FalconNodeSensor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      NodeSensorName,
					Namespace: sensorNamespacedName.Namespace,
				},
				Spec: falconv1alpha1.FalconNodeSensorSpec{
					Falcon: falconv1alpha1.FalconUnified{
						FalconSensor: falconv1alpha1.FalconSensor{
							CID: &falconCID,
						},
					},
					Node: falconv1alpha1.FalconNodeSensorConfig{
						Image: "example.com/image:test",
						ServiceAccount: falconv1alpha1.FalconNodeServiceAccount{
							Annotations: map[string]string{
								"test-annotation": "test-value",
								"another-key":     "another-value",
							},
						},
					},
					InstallNamespace: sensorNamespacedName.Namespace,
				},
			}

			err := k8sClient.Create(ctx, falconNode)
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				falconNode := &falconv1alpha1.FalconNodeSensor{}
				return k8sClient.Get(ctx, sensorNamespacedName, falconNode)
			}, 6*time.Second, time.Second).Should(Succeed())

			By("Reconciling the custom resource")
			tracker, cancel := sensorversion.NewTestTracker()
			defer cancel()

			reconciler := &FalconNodeSensorReconciler{
				Client:  k8sClient,
				Reader:  k8sReader,
				Scheme:  k8sClient.Scheme(),
				tracker: tracker,
			}

			// FalconNodeSensor is deprecated. Reconcile only adds a finalizer.
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: sensorNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking that the finalizer was added (deprecated controller only adds finalizer)")
			Eventually(func() bool {
				found := &falconv1alpha1.FalconNodeSensor{}
				if err := k8sClient.Get(ctx, sensorNamespacedName, found); err != nil {
					return false
				}
				return controllerutil.ContainsFinalizer(found, common.FalconFinalizer)
			}, 10*time.Second, time.Second).Should(BeTrue())

			By("Checking that no ServiceAccount is created via Reconcile (deprecated controller)")
			Consistently(func() bool {
				sa := &corev1.ServiceAccount{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      common.NodeServiceAccountName,
					Namespace: sensorNamespacedName.Namespace,
				}, sa)
				return errors.IsNotFound(err)
			}, 3*time.Second, time.Second).Should(BeTrue())
		})
	})
})
