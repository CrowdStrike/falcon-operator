package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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

func Dockerfile(registry, username, password string) ([]byte, error) {
	auths := dockerConfigFile{
		AuthConfigs: map[string]dockerAuthConfig{},
	}

	creds := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	newCreds := dockerAuthConfig{Auth: creds}
	auths.AuthConfigs[registry] = newCreds

	return marshal(auths)
}

func marshal(cfg dockerConfigFile) ([]byte, error) {
	file, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("Error marshaling JSON: %s", err)
	}
	return file, err
}
