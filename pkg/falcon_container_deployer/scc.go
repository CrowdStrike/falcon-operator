package falcon_container_deployer

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	securityv1 "github.com/openshift/api/security/v1"
)

const (
	SCC_NAME = "falcon-container"
)

func (d *FalconContainerDeployer) UpsertSCC() (stream *securityv1.SecurityContextConstraints, err error) {
	scc, err := d.GetSCC()
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, d.CreateSCC()
		} else if meta.IsNoMatchError(err) {
			return nil, fmt.Errorf("SecurityContextConstraints Kind is not available on the cluster: %v", err)
		}
	}
	return scc, err
}

func (d *FalconContainerDeployer) GetSCC() (*securityv1.SecurityContextConstraints, error) {
	var scc securityv1.SecurityContextConstraints
	err := d.Client.Get(d.Ctx, types.NamespacedName{Name: SCC_NAME}, &scc)
	return &scc, err
}

func (d *FalconContainerDeployer) CreateSCC() error {
	scc := &securityv1.SecurityContextConstraints{
		TypeMeta:            metav1.TypeMeta{APIVersion: securityv1.SchemeGroupVersion.String(), Kind: "SecurityContextConstraint"},
		ObjectMeta:          metav1.ObjectMeta{Name: SCC_NAME},
		AllowedCapabilities: []corev1.Capability{"SYS_PTRACE"},
	}
	if d.Instance.SCCEnabledRoot() {
		scc.RunAsUser = securityv1.RunAsUserStrategyOptions{
			Type: securityv1.RunAsUserStrategyRunAsAny,
		}
	}
	err := ctrl.SetControllerReference(d.Instance, scc, d.Scheme)
	if err != nil {
		d.Log.Error(err, "Unable to assign Controller Reference to the SecurityContextConstraints")
	}
	err = d.Client.Create(d.Ctx, scc)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			d.Log.Error(err, "Failed to create new SecurityContextConstraints", "SecurityContextConstraints.Name", scc.Name)
			return err
		}
	} else {
		d.Log.Info("Created a new SecurityContextConstraints", "SecurityContextConstraints.Name", scc.Name)
	}
	return nil
}
