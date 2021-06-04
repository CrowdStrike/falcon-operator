package falcon_container_deployer

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
)

type FalconContainerDeployer struct {
	Ctx context.Context
	client.Client
	Log      logr.Logger
	Instance *falconv1alpha1.FalconConfig
}
