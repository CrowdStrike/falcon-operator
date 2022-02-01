package falcon_container_deployer

import (
	"encoding/base64"

	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pulltoken"
)

func (d *FalconContainerDeployer) pulltoken() ([]byte, error) {
	return pulltoken.Get(d.Ctx, d.Instance.Spec.Registry.Type, d.falconApiConfig(),
		k8s_utils.QuerySecrets(d.Namespace(), d.Client),
	)
}

func (d *FalconContainerDeployer) pulltokenBase64() (string, error) {
	token, err := d.pulltoken()
	if err != nil || token == nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(token), nil
}
