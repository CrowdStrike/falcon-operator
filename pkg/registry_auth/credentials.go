package registry_auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/containers/image/v5/types"
	corev1 "k8s.io/api/core/v1"
)

// DockerCredentials manages secrets for various docker registries
type Credentials interface {
	Name() string
	DestinationContext() (*types.SystemContext, error)
	Pulltoken() (string, error)
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
		return &gcr{
			name: secret.Name,
			Key:  value,
		}
	}
	return nil
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

func (l *legacy) Pulltoken() (string, error) {
	return base64.StdEncoding.EncodeToString(l.Dockercfg), nil
}

type gcr struct {
	name string
	Key  []byte
}

func (g *gcr) Name() string {
	return g.name
}

type dockerAuthConfig struct {
	Auth string `json:"auth,omitempty"`
}

type dockerConfigFile struct {
	AuthConfigs map[string]dockerAuthConfig `json:"auths"`
}

func (g *gcr) Pulltoken() (string, error) {
	auths := dockerConfigFile{
		AuthConfigs: map[string]dockerAuthConfig{},
	}
	username := "_json_key"
	password := string(g.Key)
	creds := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	newCreds := dockerAuthConfig{Auth: creds}
	auths.AuthConfigs["gcr.io"] = newCreds

	newData, err := json.MarshalIndent(auths, "", "\t")
	if err != nil {
		return "", fmt.Errorf("Error marshaling JSON: %s", err)
	}
	return base64.StdEncoding.EncodeToString([]byte(newData)), nil
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

func (e *ecr) Pulltoken() (string, error) {
	return "", fmt.Errorf("Pulltoken on ECR not implemented")
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
