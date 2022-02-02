package falcon_container_deployer

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pulltoken"
)

const (
	JOB_SECRET_NAME = "crowdstrike-falcon-pull-secret"
)

func (d *FalconContainerDeployer) JobSecretRequired() bool {
	return d.Instance.Spec.Registry.Type == falconv1alpha1.RegistryTypeCrowdStrike
}

func (d *FalconContainerDeployer) UpsertJobSecret() (secret *corev1.Secret, err error) {
	secret, err = d.GetJobSecret()
	if err != nil && errors.IsNotFound(err) {
		return nil, d.CreateJobSecret()
	}
	return secret, err
}

func (d *FalconContainerDeployer) GetJobSecret() (*corev1.Secret, error) {
	secret := corev1.Secret{}
	err := d.Client.Get(d.Ctx, types.NamespacedName{Name: JOB_SECRET_NAME, Namespace: d.Namespace()}, &secret)
	return &secret, err
}

func (d *FalconContainerDeployer) CreateJobSecret() error {
	pulltoken, err := pulltoken.CrowdStrike(d.falconApiConfig())
	if err != nil {
		return err
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      JOB_SECRET_NAME,
			Namespace: d.Namespace(),
		},
		Data: map[string][]byte{
			".dockerconfigjson": pulltoken,
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}
	err = ctrl.SetControllerReference(d.Instance, secret, d.Scheme)
	if err != nil {
		d.Log.Error(err, "Unable to assign Controller Reference to the Job Secret")
	}
	err = d.Client.Create(d.Ctx, secret)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			d.Log.Error(err, "Failed to schedule new Job Secret", "Secret.Namespace", d.Namespace(), "Secret.Name", JOB_SECRET_NAME)
			return err
		}
	} else {
		d.Log.Info("Created a new Job Secret", "Secret.Namespace", d.Namespace(), "Secret.Name", JOB_SECRET_NAME)
	}
	return nil
}
