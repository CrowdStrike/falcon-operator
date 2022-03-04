package falcon_container_deployer

import (
	"encoding/base64"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	"github.com/crowdstrike/falcon-operator/pkg/registry"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pulltoken"
)

const (
	JOB_NAME = "falcon-configure"
)

func (d *FalconContainerDeployer) ConfigurePod() (*corev1.Pod, error) {
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(d.Instance.ObjectMeta.Namespace),
		client.MatchingLabels(map[string]string{"job-name": JOB_NAME}),
	}
	if err := d.Client.List(d.Ctx, podList, listOpts...); err != nil {
		d.Log.Error(err, "Failed to list pods", "FalconContainer.Namespace", d.Instance.ObjectMeta.Namespace, "FalconContainer.Name", d.Instance.ObjectMeta.Name)
		return nil, err
	}
	if len(podList.Items) != 1 {
		return nil, fmt.Errorf("Found %d relevant pods, expected 1 pod", len(podList.Items))
	}
	return &podList.Items[0], nil
}

func (d *FalconContainerDeployer) UpsertJob() (job *batchv1.Job, err error) {
	job, err = d.GetJob()
	if err != nil && errors.IsNotFound(err) {
		return nil, d.CreateJob()
	}
	return job, err
}

func (d *FalconContainerDeployer) GetJob() (*batchv1.Job, error) {
	job := batchv1.Job{}
	err := d.Client.Get(d.Ctx, types.NamespacedName{Name: JOB_NAME, Namespace: d.Namespace()}, &job)
	return &job, err
}

func (d *FalconContainerDeployer) CreateJob() error {
	containerSpec, err := d.installerContainer()
	if err != nil {
		return err
	}

	var pullSecrets []corev1.LocalObjectReference = nil
	if !d.imageMirroringEnabled() {
		pullSecrets = []corev1.LocalObjectReference{
			{
				Name: common.FalconPullSecretName,
			},
		}
	}

	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: batchv1.SchemeGroupVersion.String(),
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      JOB_NAME,
			Namespace: d.Namespace(),
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      JOB_NAME,
					Namespace: d.Namespace(),
				},
				Spec: corev1.PodSpec{
					RestartPolicy:    corev1.RestartPolicyOnFailure,
					Containers:       []corev1.Container{*containerSpec},
					ImagePullSecrets: pullSecrets,
				},
			},
		},
	}
	err = ctrl.SetControllerReference(d.Instance, job, d.Scheme)
	if err != nil {
		d.Log.Error(err, "Unable to assign Controller Reference to the Job")
	}
	err = d.Client.Create(d.Ctx, job)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			d.Log.Error(err, "Failed to schedule new Job", "Job.Namespace", d.Namespace(), "Job.Name", JOB_NAME)
			return err
		}
	} else {
		d.Log.Info("Created a new Job", "Job.Namespace", d.Namespace(), "Job.Name", JOB_NAME)
	}
	return nil
}

func (d *FalconContainerDeployer) installerContainer() (*corev1.Container, error) {
	imageUri, err := d.imageUri()
	if err != nil {
		return nil, err
	}
	installCmd, err := d.installerCmd(imageUri)
	if err != nil {
		return nil, err
	}

	falseP := false
	trueP := true
	return &corev1.Container{
		Name:  common.FalconInstallerJobContainerName,
		Image: imageUri,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: &falseP,
			ReadOnlyRootFilesystem:   &trueP,
		},
		Command: installCmd,
	}, nil
}

func (d *FalconContainerDeployer) installerCmd(imageUri string) ([]string, error) {
	cid, err := falcon_api.FalconCID(d.Ctx, d.Instance.Spec.FalconAPI.CID, d.falconApiConfig())
	if err != nil {
		return nil, err
	}
	installCmd := []string{"installer", "-cid", cid, "-image", imageUri}

	pulltoken, err := d.pulltokenBase64()
	if err != nil {
		return nil, err
	}
	if pulltoken != "" {
		installCmd = append(installCmd, "-pulltoken", pulltoken)
	}

	caDir := registry.CADirPath(d.Log)
	if caDir != "" {
		installCmd = append(installCmd, "-registry-certs", caDir)
	}

	return append(installCmd, d.Instance.Spec.InstallerArgs...), nil
}

func (d *FalconContainerDeployer) pulltokenBase64() (string, error) {
	if d.imageMirroringEnabled() {
		return "", nil
	}
	pulltoken, err := pulltoken.CrowdStrike(d.Ctx, d.falconApiConfig())
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pulltoken), nil
}
