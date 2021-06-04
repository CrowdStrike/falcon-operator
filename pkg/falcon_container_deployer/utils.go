package falcon_container_deployer

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
)

func (d *FalconContainerDeployer) Error(message string, err error) (ctrl.Result, error) {
	userError := fmt.Errorf("%s %w", message, err)

	d.Instance.Status.ErrorMessage = userError.Error()
	d.Instance.Status.Phase = falconv1alpha1.PhaseDone
	_ = d.Client.Status().Update(d.Ctx, d.Instance)

	return ctrl.Result{}, userError

}
