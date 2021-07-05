package falcon_container_deployer

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
)

const maxRetryAttempts = 5

func (d *FalconContainerDeployer) Error(message string, err error) (ctrl.Result, error) {
	userError := fmt.Errorf("%s %w", message, err)

	d.Instance.Status.ErrorMessage = userError.Error()

	if d.Instance.Status.RetryAttempt == nil {
		zero := uint8(0)
		d.Instance.Status.RetryAttempt = &zero
	} else if *d.Instance.Status.RetryAttempt < maxRetryAttempts {
		*(d.Instance.Status.RetryAttempt) += 1
	} else {
		d.Instance.Status.Phase = falconv1alpha1.PhaseDone
	}

	_ = d.Client.Status().Update(d.Ctx, d.Instance)

	return ctrl.Result{}, userError
}

func (d *FalconContainerDeployer) NextPhase(phase falconv1alpha1.FalconContainerStatusPhase) (ctrl.Result, error) {
	d.Instance.Status.ErrorMessage = ""
	d.Instance.Status.Phase = phase

	err := d.Client.Status().Update(d.Ctx, d.Instance)
	return ctrl.Result{}, err
}
