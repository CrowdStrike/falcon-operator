package push_auth

import (
	"github.com/containers/image/v5/types"
)

// DockerCredentials manages secrets for various docker registries
type Credentials interface {
	DestinationContext() (*types.SystemContext, error)
}

// Legacy represents old .dockercfg based credentials
type Legacy struct{}

func (l *Legacy) DestinationContext() (*types.SystemContext, error) {
	ctx := &types.SystemContext{
		LegacyFormatAuthFilePath: "/tmp/.dockercfg",
	}
	return ctx, nil
}
