package falcon

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
)

const (
	JOB_NAME = "falcon-configure"
)

func (r *FalconConfigReconciler) phaseConfiguringReconcile(ctx context.Context, instance *falconv1alpha1.FalconConfig, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Phase: Configuring")

	imageStream, err := r.imageStream(ctx, instance.ObjectMeta.Namespace)
	if err != nil {
		return r.error(ctx, instance, "Cannot access image stream", err)
	}
	falseP := false
	trueP := true
	namespace := instance.ObjectMeta.Namespace
	imageUri := imageStream.Status.DockerImageRepository
	cid := instance.Spec.FalconAPI.CID

	job := &batchv1.Job{}
	// (Step 1) Fetch Job
	err = r.Client.Get(ctx, types.NamespacedName{Name: JOB_NAME, Namespace: namespace}, job)
	if err != nil && errors.IsNotFound(err) {
		// (Step 2) create job if does not exists)
		job = &batchv1.Job{
			TypeMeta: metav1.TypeMeta{
				APIVersion: batchv1.SchemeGroupVersion.String(),
				Kind:       "Job",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      JOB_NAME,
				Namespace: namespace,
			},
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      JOB_NAME,
						Namespace: namespace,
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
		err = r.Client.Create(ctx, job)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				logger.Error(err, "Failed to schedule new Job", "Job.Namespace", namespace, "Job.Name", JOB_NAME)
				return r.error(ctx, instance, "Failed to schedule new Job", err)
			}
		}
		logger.Info("Created a new Job", "Job.Namespace", namespace, "Job.Name", JOB_NAME)
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	} else if err != nil {
		return r.error(ctx, instance, "Failed to get Job", err)
	}

	// (Step 3) verify configuration || or re-configure job
	// (Step 4) wait for completion
	// (Step 5) obtain job output
	logger.Error(nil, "TODO")

	instance.Status.ErrorMessage = ""
	instance.Status.Phase = falconv1alpha1.PhaseDone

	err = r.Client.Status().Update(ctx, instance)
	return ctrl.Result{}, err
}
