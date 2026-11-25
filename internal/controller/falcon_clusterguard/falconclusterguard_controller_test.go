package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
)

var _ = Describe("FalconClusterGuard Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-resource"
			resourceNamespace = "default"
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}
		falconclusterguard := &falconv1alpha1.FalconClusterGuard{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind FalconClusterGuard")
			err := k8sClient.Get(ctx, typeNamespacedName, falconclusterguard)
			if err != nil && errors.IsNotFound(err) {
				resource := &falconv1alpha1.FalconClusterGuard{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: resourceNamespace,
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &falconv1alpha1.FalconClusterGuard{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance FalconClusterGuard")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &FalconClusterGuardReconciler{
				Client:        k8sClient,
				RuntimeScheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			// Without Falcon.CID or FalconAPI credentials the reconciler returns an error
			// rather than panicking; verify it is a CID resolution error.
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot resolve CID"))
		})
	})

	Context("When validating the guardian proxy configuration", func() {
		ctx := context.Background()

		newClusterGuard := func(name string, proxy falconv1alpha1.FalconClusterGuardGuardianProxy) *falconv1alpha1.FalconClusterGuard {
			resource := &falconv1alpha1.FalconClusterGuard{
				ObjectMeta: metav1.ObjectMeta{Name: name},
			}
			resource.Spec.NodeSensor.Guardian.Proxy = proxy
			return resource
		}

		It("should reject an enabled proxy without a port", func() {
			enabled := true
			resource := newClusterGuard("guardian-proxy-no-port", falconv1alpha1.FalconClusterGuardGuardianProxy{
				Enabled: &enabled,
			})
			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(errors.IsInvalid(err)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("port is required when the guardian proxy is enabled"))
		})

		It("should accept an enabled proxy with a port", func() {
			enabled := true
			port := int32(48080)
			resource := newClusterGuard("guardian-proxy-with-port", falconv1alpha1.FalconClusterGuardGuardianProxy{
				Enabled: &enabled,
				Port:    &port,
			})
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should accept a disabled proxy without a port", func() {
			enabled := false
			resource := newClusterGuard("guardian-proxy-disabled", falconv1alpha1.FalconClusterGuardGuardianProxy{
				Enabled: &enabled,
			})
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
	})

	Context("When validating the internal build stage", func() {
		ctx := context.Background()

		It("should default the build stage to release", func() {
			resource := &falconv1alpha1.FalconClusterGuard{
				ObjectMeta: metav1.ObjectMeta{Name: "build-stage-default"},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			Expect(resource.Spec.Internal.BuildStage).To(Equal(falconv1alpha1.BuildStageRelease))
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should accept the stage build stage", func() {
			resource := &falconv1alpha1.FalconClusterGuard{
				ObjectMeta: metav1.ObjectMeta{Name: "build-stage-stage"},
			}
			resource.Spec.Internal.BuildStage = falconv1alpha1.BuildStageStage
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			Expect(resource.Spec.Internal.BuildStage).To(Equal(falconv1alpha1.BuildStageStage))
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should reject an unknown build stage", func() {
			resource := &falconv1alpha1.FalconClusterGuard{
				ObjectMeta: metav1.ObjectMeta{Name: "build-stage-unknown"},
			}
			resource.Spec.Internal.BuildStage = "nightly"
			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(errors.IsInvalid(err)).To(BeTrue())
		})
	})
})
