package v1alpha1

import (
	"github.com/crowdstrike/gofalcon/falcon"
)

// ApiConfig generates standard gofalcon library api config
func (fa *FalconAPI) ApiConfig() *falcon.ApiConfig {
	return &falcon.ApiConfig{
		Cloud:        falcon.Cloud(fa.CloudRegion),
		ClientId:     fa.ClientId,
		ClientSecret: fa.ClientSecret,
	}
}
