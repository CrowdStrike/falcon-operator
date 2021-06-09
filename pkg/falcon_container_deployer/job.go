package falcon_container_deployer

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
)

const (
	JOB_NAME = "falcon-configure"
)

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
	imageStream, err := d.GetImageStream()
	if err != nil {
		return err
	}

	falseP := false
	trueP := true
	cid := d.Instance.Spec.FalconAPI.CID
	imageUri := imageStream.Status.DockerImageRepository

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
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:  "installer",
							Image: imageUri,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: &falseP,
								ReadOnlyRootFilesystem:   &trueP,
							},
							Command: []string{
								"installer",
								"-cid", cid,
								"-image", imageUri,
							},
						},
					},
				},
			},
		},
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
