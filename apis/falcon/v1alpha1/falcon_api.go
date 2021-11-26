package v1alpha1

import (
	"context"
	"fmt"

	"github.com/crowdstrike/gofalcon/falcon"
)

// ApiConfig generates standard gofalcon library api config
func (fa *FalconAPI) ApiConfig() *falcon.ApiConfig {
	return &falcon.ApiConfig{
		Cloud:        fa.falconCloudInternal(),
		ClientId:     fa.ClientId,
		ClientSecret: fa.ClientSecret,
	}
}

func (fa *FalconAPI) FalconCloud(ctx context.Context) (falcon.CloudType, error) {
	cloud := fa.falconCloudInternal()
	err := cloud.Autodiscover(ctx, fa.ClientId, fa.ClientSecret)
	if err != nil {
		return cloud, fmt.Errorf("Could not autodiscover Falcon Cloud Region. Please provide your cloud_region in FalconContainer Spec: %v", err)
	}
	return cloud, nil
}

func (fa *FalconAPI) falconCloudInternal() falcon.CloudType {
	if fa.CloudRegion == "autodiscover" {
		return falcon.Cloud("")
	} else {
		return falcon.Cloud(fa.CloudRegion)
	}
}
