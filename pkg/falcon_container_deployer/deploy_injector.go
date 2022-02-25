package falcon_container_deployer

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
)

func (d *FalconContainerDeployer) deployInjector(objects []runtime.Object) error {
	if d.Instance.Spec.Injector == nil || d.Instance.Spec.Injector.SAAnnotations == nil || len(d.Instance.Spec.Injector.SAAnnotations) == 0 {
		return k8s_utils.Create(d.Ctx, d.Client, objects, d.Log)
	}

	nsObject, otherObjects := k8s_utils.PopNamespaceFromObjectList(objects)

	err := k8s_utils.Create(d.Ctx, d.Client, []runtime.Object{nsObject}, d.Log)
	if err != nil {
		return err
	}
	err = d.patchInjectorServiceAccount()
	if err != nil {
		return err
	}

	return k8s_utils.Create(d.Ctx, d.Client, otherObjects, d.Log)
}

func (d *FalconContainerDeployer) patchInjectorServiceAccount() error {
	sa, err := d.getInjectorSa()
	if err != nil {
		return err
	}
	patch := client.MergeFrom(sa.DeepCopy())

	if sa.ObjectMeta.Annotations == nil {
		sa.ObjectMeta.Annotations = map[string]string{}
	}
	for k, v := range d.Instance.Spec.Injector.SAAnnotations {
		d.Log.Info("Patching injector service account: adding annotation", k, v)
		sa.ObjectMeta.Annotations[k] = v
	}

	return d.Client.Patch(d.Ctx, sa, patch)
}

func (d *FalconContainerDeployer) getInjectorSa() (*corev1.ServiceAccount, error) {
	sa := corev1.ServiceAccount{}
	err := d.Client.Get(d.Ctx, types.NamespacedName{Name: "default", Namespace: "falcon-system"}, &sa)
	return &sa, err
}
