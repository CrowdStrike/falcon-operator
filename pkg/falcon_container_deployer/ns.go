package falcon_container_deployer

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
)

const (
	NAMESPACE = "falcon-system-configure"
)

func (d *FalconContainerDeployer) UpsertNamespace(namespace string) (ns *corev1.Namespace, err error) {
	ns, err = d.GetNamespace(namespace)
	if err != nil && errors.IsNotFound(err) {
		return nil, d.CreateNamespace(namespace)
	}
	return ns, err
}

func (d *FalconContainerDeployer) GetNamespace(namespace string) (*corev1.Namespace, error) {
	ns := corev1.Namespace{}
	err := d.Client.Get(d.Ctx, types.NamespacedName{Name: namespace}, &ns)
	return &ns, err
}

func (d *FalconContainerDeployer) CreateNamespace(namespace string) error {
	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	err := d.Client.Create(d.Ctx, ns)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			d.Log.Error(err, "Failed to schedule new namespace", "Namespace.Name", namespace)
			return err
		}
	} else {
		d.Log.Info("Created a new namespace", "Namespace.Name", namespace)
	}
	return nil
}

// Returns the namespace in which the operator runs and creates helper artifacts
func (d *FalconContainerDeployer) Namespace() string {
	return NAMESPACE
}
