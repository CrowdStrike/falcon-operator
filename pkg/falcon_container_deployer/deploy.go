package falcon_container_deployer

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
)

type FalconContainerDeployer struct {
	Ctx context.Context
	client.Client
	Log      logr.Logger
	Instance *falconv1alpha1.FalconConfig
}

func (d *FalconContainerDeployer) PhasePending() (ctrl.Result, error) {
	stream, err := d.UpsertImageStream()
	if err != nil {
		return d.Error("failed to upsert Image Stream", err)
	}
	if stream == nil {
		// It takes few moment for the ImageStream to be ready (shortly after it has been created)
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	return d.NextPhase(falconv1alpha1.PhaseBuilding)
}

func (d *FalconContainerDeployer) PhaseBuilding() (ctrl.Result, error) {
	err := d.EnsureDockercfg()
	if err != nil {
		return d.Error("Cannot find dockercfg secret from the current namespace", err)
	}
	err = d.PushImage()
	if err != nil {
		return d.Error("Cannot refresh Falcon Container image", err)
	}

	return d.NextPhase(falconv1alpha1.PhaseConfiguring)
}
