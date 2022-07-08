package common

import (
	"context"
	"fmt"

	"github.com/crowdstrike/falcon-operator/pkg/registry/falcon_registry"
	"github.com/crowdstrike/gofalcon/falcon"
)

func ImageInfo(ctx context.Context, imageVer *string, fa *falcon.ApiConfig) (string, string, error) {
	fa.Context = ctx
	if imageVer != nil && *imageVer != "" {
		return *imageVer, "", nil
	}
	registry, err := falcon_registry.NewFalconRegistry(ctx, fa)
	if err != nil {
		return "", "", err
	}

	tag, err := registry.LastContainerTag(ctx, imageVer)
	if err == nil {
		imageVer = &tag
	}

	falconUri := ImageURI(fa.Cloud)

	return falconUri, tag, err
}

func ImageURI(falconCloud falcon.CloudType) string {
	return fmt.Sprintf("registry.crowdstrike.com/falcon-container/%s/release/falcon-sensor", falconCloud.String())
}
