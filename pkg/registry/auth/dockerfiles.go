package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
)

type dockerAuthConfig struct {
	Auth string `json:"auth,omitempty"`
}

type dockerConfigFile struct {
	AuthConfigs map[string]dockerAuthConfig `json:"auths"`
}

func dockerJsonValid(raw []byte) bool {
	content, err := parseConfig(raw)
	return (err == nil && len(content.AuthConfigs) != 0)
}

func parseConfig(raw []byte) (result dockerConfigFile, err error) {
	err = json.Unmarshal(raw, &result)
	return
}

func parse(raw []byte, legacyFormat bool) (result dockerConfigFile, err error) {
	if legacyFormat {
		err = json.Unmarshal(raw, &result.AuthConfigs)
		return result, err
	} else {
		return parseConfig(raw)
	}
}

func Dockerfile(registry, username, password string) ([]byte, error) {
	auths := dockerConfigFile{
		AuthConfigs: map[string]dockerAuthConfig{},
	}

	creds := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	newCreds := dockerAuthConfig{Auth: creds}
	auths.AuthConfigs[registry] = newCreds

	return marshal(auths)
}

func MergeCredentials(credentials []Credentials, log logr.Logger) ([]byte, error) {
	merged := dockerConfigFile{
		AuthConfigs: map[string]dockerAuthConfig{},
	}

	for _, creds := range credentials {
		pulltoken, err := creds.Pulltoken()
		if err != nil {
			log.Error(err, fmt.Sprintf("Cannot parse docker config secret '%s'. Skipping. It won't be forwarded to Falcon Injector.", creds.Name()))
		}

		parsed, err := parse(pulltoken, creds.legacy())
		if err != nil {
			log.Error(err, fmt.Sprintf("Cannot parse docker config secret '%s'. Skipping. It won't be forwarded to Falcon Injector.", creds.Name()))
		}

		log.Info("Found pull secret to be forwarded to Falcon Container Injector: ", "secret.Name", creds.Name())

		for k, v := range parsed.AuthConfigs {
			merged.AuthConfigs[k] = v
		}
	}
	return marshal(merged)
}

func marshal(cfg dockerConfigFile) ([]byte, error) {
	file, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("Error marshaling JSON: %s", err)
	}
	return file, err
}
