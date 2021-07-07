package push_auth

import (
	"io/ioutil"

	"github.com/containers/image/v5/types"
	corev1 "k8s.io/api/core/v1"
)

// DockerCredentials manages secrets for various docker registries
type Credentials interface {
	Name() string
	DestinationContext() (*types.SystemContext, error)
}

func GetCredentials(secrets []corev1.Secret) Credentials {
	for _, secret := range secrets {
		if secret.Data == nil {
			continue
		}
		if secret.Type != "kubernetes.io/dockercfg" && secret.Type != "kubernetes.io/dockerconfigjson" {
			continue
		}

		if (secret.ObjectMeta.Annotations == nil || secret.ObjectMeta.Annotations["kubernetes.io/service-account.name"] != "builder") && secret.Name != "builder" {
			continue
		}

		value, ok := secret.Data[".dockercfg"]
		if ok {
			return &legacy{
				name:      secret.Name,
				Dockercfg: value,
			}
		}
		value, ok = secret.Data[".dockerconfigjson"]
		if ok {
			return &gcr{
				name: secret.Name,
				Key:  value,
			}
		}
	}
	return nil
}

// Legacy represents old .dockercfg based credentials
type legacy struct {
	name      string
	Dockercfg []byte
}

func (l *legacy) DestinationContext() (*types.SystemContext, error) {
	const dockerCfgFile = "/tmp/.dockercfg"

	err := ioutil.WriteFile(dockerCfgFile, l.Dockercfg, 0600)
	if err != nil {
		return nil, err
	}

	ctx := &types.SystemContext{
		LegacyFormatAuthFilePath: dockerCfgFile,
	}
	return ctx, nil
}

func (l *legacy) Name() string {
	return l.name
}

type gcr struct {
	name string
	Key  []byte
}

func (g *gcr) DestinationContext() (*types.SystemContext, error) {
	return &types.SystemContext{
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: "_json_key",
			Password: string(g.Key),
		},
	}, nil
}

func (g *gcr) Name() string {
	return g.name
}
