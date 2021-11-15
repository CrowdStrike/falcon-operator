package v1alpha1

import (
	"github.com/crowdstrike/gofalcon/falcon"
)

// ApiConfig generates standard gofalcon library api config
func (fa *FalconAPI) ApiConfig() *falcon.ApiConfig {
	return &falcon.ApiConfig{
		Cloud:        fa.FalconCloud(),
		ClientId:     fa.ClientId,
		ClientSecret: fa.ClientSecret,
	}
}

func (fa *FalconAPI) FalconCloud() falcon.CloudType {
	return falcon.Cloud(fa.CloudRegion)
}
