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
})
