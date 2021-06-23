package push_auth

import (
	"io/ioutil"

	"github.com/containers/image/v5/types"
)

// DockerCredentials manages secrets for various docker registries
type Credentials interface {
	DestinationContext() (*types.SystemContext, error)
}

// Legacy represents old .dockercfg based credentials
type Legacy struct {
	Dockercfg []byte
}

func (l *Legacy) DestinationContext() (*types.SystemContext, error) {
	err := ioutil.WriteFile("/tmp/.dockercfg", l.Dockercfg, 0600)
	if err != nil {
		return nil, err
	}

	ctx := &types.SystemContext{
		LegacyFormatAuthFilePath: "/tmp/.dockercfg",
	}
	return ctx, nil
}
