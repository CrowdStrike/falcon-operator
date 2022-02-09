package falcon_container_deployer

import (
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
)

func (d *FalconContainerDeployer) deployInjector(objects []runtime.Object) error {
	return k8s_utils.Create(d.Ctx, d.Client, objects, d.Log)
}
