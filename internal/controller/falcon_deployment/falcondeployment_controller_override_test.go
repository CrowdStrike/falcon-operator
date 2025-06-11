package falcon

import (
	"context"
	"fmt"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("FalconDeployment Controller", func() {
	Context("FalconDeployment controller no override test", func() {
		FalconDeploymentName := "test-falcondeployment"
		FalconDeploymentOverrideName := "test-falcondeployment-override"
		falconClientID := "12345678912345678912345678912345"
		falconSecret := "32165498732165498732165498732165498732165"
		falconCID := "1234567890ABCDEF1234567890ABCDEF-12"
		falconCloudRegion := "us-1"
		defaultRegistryName := "crowdstrike"
		overrideRegistryName := "ecr"
		overrideFalconClientID := "11111111111111111111111111111111"
		overrideFalconSecret := "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"
		overrideFalconCID := "11111111111111111111111111111111-FF"
		overrideFalconCloudRegion := "us-2"
		sidecarSensorNamespace := "falcon-system"
		sidecarSensorOverrideNamespace := "falcon-system-override"
		sidecarSensorNamespacedName := "falcon-container-sensor"
		nodeSensorNamespace := "falcon-node-sensor"
		nodeSensorOverrideNamespace := "falcon-node-sensor-override"
		nodeSensorNamespacedName := "falcon-node-sensor"
		imageAnalyzerNamespace := "falcon-iar"
		imageAnalyzerOverrideNamespace := "falcon-iar-override"
		imageAnalyzerNamespacedName := "falcon-image-analyzer"
		admissionControllerNamespace := "falcon-kac"
		admissionControllerOverrideNamespace := "falcon-kac-override"
		admissionControllerNamespacedName := "falcon-kac"

		ctx := context.Background()

		falconDeploymentNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      FalconDeploymentName,
				Namespace: FalconDeploymentName,
			},
		}

		falconDeploymentOverrideNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      FalconDeploymentOverrideName,
				Namespace: FalconDeploymentOverrideName,
			},
		}

		BeforeEach(func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, falconDeploymentNamespace)
			Expect(err).To(Not(HaveOccurred()))

			By("Creating the Override Namespace to perform the tests")
			err = k8sClient.Create(ctx, falconDeploymentOverrideNamespace)
			Expect(err).To(Not(HaveOccurred()))
		})

		AfterEach(func() {
			// TODO(user): Attention if you improve this code by adding other context test you MUST
			// be aware of the current delete namespace limitations. More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Deleting the Namespace to perform the tests")
			_ = k8sClient.Delete(ctx, falconDeploymentNamespace)
			By("Deleting the Override Namespace to perform the tests")
			_ = k8sClient.Delete(ctx, falconDeploymentOverrideNamespace)
		})

		typeContainerNamespacedName := types.NamespacedName{Name: sidecarSensorNamespacedName, Namespace: sidecarSensorNamespace}
		typeAdmissionNamespacedName := types.NamespacedName{Name: admissionControllerNamespacedName, Namespace: admissionControllerNamespace}
		typeIARNamespacedName := types.NamespacedName{Name: imageAnalyzerNamespacedName, Namespace: imageAnalyzerNamespace}
		typeNodeNamespacedName := types.NamespacedName{Name: nodeSensorNamespacedName, Namespace: nodeSensorNamespace}
		typeNamespacedName := types.NamespacedName{Name: FalconDeploymentName, Namespace: FalconDeploymentName}

		typeContainerNamespacedOverrideName := types.NamespacedName{Name: sidecarSensorNamespacedName, Namespace: sidecarSensorOverrideNamespace}
		typeAdmissionNamespacedOverrideName := types.NamespacedName{Name: admissionControllerNamespacedName, Namespace: admissionControllerOverrideNamespace}
		typeIARNamespacedOverrideName := types.NamespacedName{Name: imageAnalyzerNamespacedName, Namespace: imageAnalyzerOverrideNamespace}
		typeNodeNamespacedOverrideName := types.NamespacedName{Name: nodeSensorNamespacedName, Namespace: nodeSensorOverrideNamespace}
		typeNamespacedOverrideName := types.NamespacedName{Name: FalconDeploymentOverrideName, Namespace: FalconDeploymentOverrideName}

		defaultRegistry := falconv1alpha1.RegistrySpec{
			Type: falconv1alpha1.RegistryTypeSpec(defaultRegistryName),
		}

		overrideRegistry := falconv1alpha1.RegistrySpec{
			Type: falconv1alpha1.RegistryTypeSpec(overrideRegistryName),
		}

		mockFalconAPI := falconv1alpha1.FalconAPI{
			CloudRegion:  falconCloudRegion,
			ClientId:     falconClientID,
			ClientSecret: falconSecret,
			CID:          &falconCID,
		}

		overrideFalconAPI := falconv1alpha1.FalconAPI{
			CloudRegion:  overrideFalconCloudRegion,
			ClientId:     overrideFalconClientID,
			ClientSecret: overrideFalconSecret,
			CID:          &overrideFalconCID,
		}

		falconSecretNameSpace := "falcon-secrets"
		falconSecretName := "falcon-secret"
		overrideFalconSecretNameSpace := "falcon-secrets-override"
		overrideFalconSecretName := "falcon-secret-override"

		topLevelFalconSecret := falconv1alpha1.FalconSecret{
			Namespace:  falconSecretNameSpace,
			SecretName: falconSecretName,
		}

		lowerLevelFalconSecret := falconv1alpha1.FalconSecret{
			Namespace:  overrideFalconSecretNameSpace,
			SecretName: overrideFalconSecretName,
		}

		It("should successfully reconcile the resource", func() {
			By("Creating the custom resource for the Kind FalconDeployment - No Overrides")
			deployContainerSensor := true

			falconDeployment := &falconv1alpha1.FalconDeployment{}
			err := k8sClient.Get(ctx, typeNamespacedName, falconDeployment)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				falconDeployment := &falconv1alpha1.FalconDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      FalconDeploymentName,
						Namespace: falconDeploymentNamespace.Name,
					},
					Spec: falconv1alpha1.FalconDeploymentSpec{
						FalconAPI:             &mockFalconAPI,
						Registry:              defaultRegistry,
						FalconSecret:          topLevelFalconSecret,
						DeployContainerSensor: &deployContainerSensor,
						FalconAdmission: falconv1alpha1.FalconAdmissionSpec{
							InstallNamespace: admissionControllerNamespace,
							Registry:         defaultRegistry,
						},
						FalconNodeSensor: falconv1alpha1.FalconNodeSensorSpec{
							InstallNamespace: nodeSensorNamespace,
						},
						FalconImageAnalyzer: falconv1alpha1.FalconImageAnalyzerSpec{
							InstallNamespace: imageAnalyzerNamespace,
							Registry:         defaultRegistry,
						},
						FalconContainerSensor: falconv1alpha1.FalconContainerSpec{
							InstallNamespace: sidecarSensorNamespace,
							Registry:         defaultRegistry,
						},
					},
				}

				err = k8sClient.Create(ctx, falconDeployment)
				Expect(err).To(Not(HaveOccurred()))
			}

			By("Checking if FalconDeployment was created")
			Eventually(func() error {
				found := &falconv1alpha1.FalconDeployment{}
				return k8sClient.Get(ctx, typeNamespacedName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Reconciling the FalconDeployment custom resource created")
			falconDeploymentReconciler := &FalconDeploymentReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = falconDeploymentReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Validate FalconDeployment top level FalconAPI credentials are used in the child CRs - FalconAdmission - without overrides")
			falconAdmission := &falconv1alpha1.FalconAdmission{}
			err = k8sClient.Get(ctx, typeAdmissionNamespacedName, falconAdmission)
			Expect(err).To(Not(HaveOccurred()))
			Expect(falconAdmission.Spec.FalconAPI.ClientId).To(Equal(mockFalconAPI.ClientId))
			Expect(falconAdmission.Spec.FalconAPI.ClientSecret).To(Equal(mockFalconAPI.ClientSecret))
			Expect(falconAdmission.Spec.FalconAPI.CloudRegion).To(Equal(mockFalconAPI.CloudRegion))

			By("Validate FalconDeployment top level FalconAPI credentials are used in the child CRs - FalconImageAnalyzer - without overrides")
			falconImageAnalyzer := &falconv1alpha1.FalconImageAnalyzer{}
			err = k8sClient.Get(ctx, typeIARNamespacedName, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))
			Expect(falconImageAnalyzer.Spec.FalconAPI.ClientId).To(Equal(mockFalconAPI.ClientId))
			Expect(falconImageAnalyzer.Spec.FalconAPI.ClientSecret).To(Equal(mockFalconAPI.ClientSecret))
			Expect(falconImageAnalyzer.Spec.FalconAPI.CloudRegion).To(Equal(mockFalconAPI.CloudRegion))

			By("Validate FalconDeployment top level FalconAPI credentials are used in the child CRss - FalconNodeSensor - without overrides")
			falconNodeSensor := &falconv1alpha1.FalconNodeSensor{}
			err = k8sClient.Get(ctx, typeNodeNamespacedName, falconNodeSensor)
			Expect(err).To(Not(HaveOccurred()))
			Expect(falconNodeSensor.Spec.FalconAPI.ClientId).To(Equal(mockFalconAPI.ClientId))
			Expect(falconNodeSensor.Spec.FalconAPI.ClientSecret).To(Equal(mockFalconAPI.ClientSecret))
			Expect(falconNodeSensor.Spec.FalconAPI.CloudRegion).To(Equal(mockFalconAPI.CloudRegion))

			By("Validate FalconDeployment top level FalconAPI credentials are used in the child CRs - FalconContainer - without overrides")
			falconContainer := &falconv1alpha1.FalconContainer{}
			err = k8sClient.Get(ctx, typeContainerNamespacedName, falconContainer)
			Expect(err).To(Not(HaveOccurred()))
			Expect(falconContainer.Spec.FalconAPI.ClientId).To(Equal(mockFalconAPI.ClientId))
			Expect(falconContainer.Spec.FalconAPI.ClientSecret).To(Equal(mockFalconAPI.ClientSecret))
			Expect(falconContainer.Spec.FalconAPI.CloudRegion).To(Equal(mockFalconAPI.CloudRegion))

			By("Validate FalconDeployment top level FalconSecret spec is used in the child CRs - without overrides")
			Expect(falconAdmission.Spec.FalconSecret).To(Equal(topLevelFalconSecret))
			Expect(falconImageAnalyzer.Spec.FalconSecret).To(Equal(topLevelFalconSecret))
			Expect(falconNodeSensor.Spec.FalconSecret).To(Equal(topLevelFalconSecret))
			Expect(falconContainer.Spec.FalconSecret).To(Equal(topLevelFalconSecret))

			By("Deleting the FalconDeployment to perform the tests")
			_ = k8sClient.Delete(ctx, falconAdmission)
			_ = k8sClient.Delete(ctx, falconImageAnalyzer)
			_ = k8sClient.Delete(ctx, falconNodeSensor)
			_ = k8sClient.Delete(ctx, falconContainer)

			By("Creating the custom resource for the Kind FalconDeployment - With Overrides")
			deployContainerSensor = true

			overrideFalconDeployment := &falconv1alpha1.FalconDeployment{}
			err = k8sClient.Get(ctx, typeNamespacedOverrideName, overrideFalconDeployment)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				falconDeployment := &falconv1alpha1.FalconDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      FalconDeploymentOverrideName,
						Namespace: falconDeploymentOverrideNamespace.Name,
					},
					Spec: falconv1alpha1.FalconDeploymentSpec{
						FalconAPI:             &mockFalconAPI,
						Registry:              defaultRegistry,
						DeployContainerSensor: &deployContainerSensor,
						FalconAdmission: falconv1alpha1.FalconAdmissionSpec{
							InstallNamespace: admissionControllerOverrideNamespace,
							FalconAPI:        &overrideFalconAPI,
							Registry:         overrideRegistry,
							FalconSecret:     lowerLevelFalconSecret,
						},
						FalconNodeSensor: falconv1alpha1.FalconNodeSensorSpec{
							InstallNamespace: nodeSensorOverrideNamespace,
							FalconAPI:        &overrideFalconAPI,
							FalconSecret:     lowerLevelFalconSecret,
						},
						FalconImageAnalyzer: falconv1alpha1.FalconImageAnalyzerSpec{
							InstallNamespace: imageAnalyzerOverrideNamespace,
							FalconAPI:        &overrideFalconAPI,
							Registry:         overrideRegistry,
							FalconSecret:     lowerLevelFalconSecret,
						},
						FalconContainerSensor: falconv1alpha1.FalconContainerSpec{
							InstallNamespace: sidecarSensorOverrideNamespace,
							FalconAPI:        &overrideFalconAPI,
							Registry:         overrideRegistry,
							FalconSecret:     lowerLevelFalconSecret,
						},
					},
				}

				err = k8sClient.Create(ctx, falconDeployment)
				Expect(err).To(Not(HaveOccurred()))
			}

			By("Checking if FalconDeployment was created - With Overrides")
			Eventually(func() error {
				found := &falconv1alpha1.FalconDeployment{}
				return k8sClient.Get(ctx, typeNamespacedOverrideName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Reconciling the FalconDeployment custom resource created - With Overrides")
			_, err = falconDeploymentReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedOverrideName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Validate override FalconAPI credentials are used in the child CRs - FalconAdmission - With Overrides")
			falconAdmission = &falconv1alpha1.FalconAdmission{}
			err = k8sClient.Get(ctx, typeAdmissionNamespacedOverrideName, falconAdmission)
			fmt.Println(falconAdmission.Spec.FalconAPI)
			Expect(err).To(Not(HaveOccurred()))
			Expect(falconAdmission.Spec.FalconAPI.ClientId).To(Equal(overrideFalconAPI.ClientId))
			Expect(falconAdmission.Spec.FalconAPI.ClientSecret).To(Equal(overrideFalconAPI.ClientSecret))
			Expect(falconAdmission.Spec.FalconAPI.CloudRegion).To(Equal(overrideFalconAPI.CloudRegion))

			By("Validate override FalconAPI credentials are used in the child CRs - FalconImageAnalyzer - With Overrides")
			falconImageAnalyzer = &falconv1alpha1.FalconImageAnalyzer{}
			err = k8sClient.Get(ctx, typeIARNamespacedOverrideName, falconImageAnalyzer)
			Expect(err).To(Not(HaveOccurred()))
			Expect(falconImageAnalyzer.Spec.FalconAPI.ClientId).To(Equal(overrideFalconAPI.ClientId))
			Expect(falconImageAnalyzer.Spec.FalconAPI.ClientSecret).To(Equal(overrideFalconAPI.ClientSecret))
			Expect(falconImageAnalyzer.Spec.FalconAPI.CloudRegion).To(Equal(overrideFalconAPI.CloudRegion))

			By("Validate override FalconAPI credentials are used in the child CRss - FalconNodeSensor - With Overrides")
			falconNodeSensor = &falconv1alpha1.FalconNodeSensor{}
			err = k8sClient.Get(ctx, typeNodeNamespacedOverrideName, falconNodeSensor)
			Expect(err).To(Not(HaveOccurred()))
			Expect(falconNodeSensor.Spec.FalconAPI.ClientId).To(Equal(overrideFalconAPI.ClientId))
			Expect(falconNodeSensor.Spec.FalconAPI.ClientSecret).To(Equal(overrideFalconAPI.ClientSecret))
			Expect(falconNodeSensor.Spec.FalconAPI.CloudRegion).To(Equal(overrideFalconAPI.CloudRegion))

			By("Validate override FalconAPI credentials are used in the child CRs - FalconContainer - With Overrides")
			falconContainer = &falconv1alpha1.FalconContainer{}
			err = k8sClient.Get(ctx, typeContainerNamespacedOverrideName, falconContainer)
			Expect(err).To(Not(HaveOccurred()))
			Expect(falconContainer.Spec.FalconAPI.ClientId).To(Equal(overrideFalconAPI.ClientId))
			Expect(falconContainer.Spec.FalconAPI.ClientSecret).To(Equal(overrideFalconAPI.ClientSecret))
			Expect(falconContainer.Spec.FalconAPI.CloudRegion).To(Equal(overrideFalconAPI.CloudRegion))

			By("Validate lower level FalconSecret spec is used in the child CRs - with overrides")
			Expect(falconAdmission.Spec.FalconSecret).To(Equal(lowerLevelFalconSecret))
			Expect(falconImageAnalyzer.Spec.FalconSecret).To(Equal(lowerLevelFalconSecret))
			Expect(falconNodeSensor.Spec.FalconSecret).To(Equal(lowerLevelFalconSecret))
			Expect(falconContainer.Spec.FalconSecret).To(Equal(lowerLevelFalconSecret))
		})
	})
})
