package falcon_container_deployer

import (
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
)

func (d *FalconContainerDeployer) imageMirroringEnabled() bool {
	return d.Instance.Spec.Registry.Type != falconv1alpha1.RegistryTypeCrowdStrike
}
