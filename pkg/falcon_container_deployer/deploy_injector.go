package falcon_container_deployer

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
)

func (d *FalconContainerDeployer) deployInjector(objects []runtime.Object) error {
	if d.Instance.Spec.Registry.EcrIamRoleArnForInjector == nil {
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
	sa.ObjectMeta.Annotations["eks.amazonaws.com/role-arn"] = *d.Instance.Spec.Registry.EcrIamRoleArnForInjector

	d.Log.Info("Patching injector service account: adding ECR Role", "EcrRoleArn", *d.Instance.Spec.Registry.EcrIamRoleArnForInjector)
	return d.Client.Patch(d.Ctx, sa, patch)
}

func (d *FalconContainerDeployer) getInjectorSa() (*corev1.ServiceAccount, error) {
	sa := corev1.ServiceAccount{}
	err := d.Client.Get(d.Ctx, types.NamespacedName{Name: "default", Namespace: "falcon-system"}, &sa)
	return &sa, err
}
