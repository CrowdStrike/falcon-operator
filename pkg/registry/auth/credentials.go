package auth

import (
	"fmt"
	"os"

	"go.podman.io/image/v5/types"
	corev1 "k8s.io/api/core/v1"
)

// DockerCredentials manages secrets for various docker registries
type Credentials interface {
	Name() string
	DestinationContext() (*types.SystemContext, error)
	Pulltoken() ([]byte, error)
}

func newCreds(secret corev1.Secret) Credentials {
	value, ok := secret.Data[".dockercfg"]
	if ok {
		return &legacy{
			name:      secret.Name,
			Dockercfg: value,
		}
	}
	value, ok = secret.Data[".dockerconfigjson"]
	if ok {
		if dockerJsonValid(value) {
			return &classic{
				name:  secret.Name,
				value: value,
			}
		}

		return &gcr{
			name: secret.Name,
			Key:  value,
		}
	}
	return nil
}

func GetPushCredentials(secrets []corev1.Secret) Credentials {
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

		creds := newCreds(secret)
		if creds != nil {
			return creds
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

	err := os.WriteFile(dockerCfgFile, l.Dockercfg, 0600)
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

func (l *legacy) Pulltoken() ([]byte, error) {
	return l.Dockercfg, nil
}

type classic struct {
	name  string
	value []byte
}

func (c *classic) Name() string {
	return c.name
}

func (c *classic) DestinationContext() (*types.SystemContext, error) {

	const dockerCfgFile = "/tmp/.dockercfg"

	err := os.WriteFile(dockerCfgFile, c.value, 0600)
	if err != nil {
		return nil, err
	}
	return &types.SystemContext{
		AuthFilePath: dockerCfgFile,
	}, nil
}

func (c *classic) Pulltoken() ([]byte, error) {
	return c.value, nil
}

type gcr struct {
	name string
	Key  []byte
}

func (g *gcr) Name() string {
	return g.name
}

func (g *gcr) Pulltoken() ([]byte, error) {
	username := "_json_key"
	password := string(g.Key)
	newData, err := Dockerfile("gcr.io", username, password)
	if err != nil {
		return nil, fmt.Errorf("Could not create pull token for GCR: %s", err)
	}
	return newData, nil
}

func (g *gcr) DestinationContext() (*types.SystemContext, error) {
	return &types.SystemContext{
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: "_json_key",
			Password: string(g.Key),
		},
	}, nil
}

type ecr struct {
	password string
}

func (e *ecr) Name() string {
	return "ECR Token from AWS API"
}

func (e *ecr) Pulltoken() ([]byte, error) {
	return nil, fmt.Errorf("Pulltoken on ECR not implemented")
}

func (e *ecr) DestinationContext() (*types.SystemContext, error) {
	return &types.SystemContext{
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: "AWS",
			Password: e.password,
		},
	}, nil
}

func ECRCredentials(token string) (Credentials, error) {
	if token[0:4] != "AWS:" {
		return nil, fmt.Errorf("Could not parse EKS crendentials token. Expected to start with 'AWS:', got: '%s'", token[0:4])
	}
	return &ecr{
		password: token[4:],
	}, nil
}
