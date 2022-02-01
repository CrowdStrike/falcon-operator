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
	var content dockerConfigFile
	err := json.Unmarshal(raw, &content)
	return (err == nil && len(content.AuthConfigs) != 0)
}

func Dockerfile(registry, username, password string) ([]byte, error) {
	auths := dockerConfigFile{
		AuthConfigs: map[string]dockerAuthConfig{},
	}

	creds := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	newCreds := dockerAuthConfig{Auth: creds}
	auths.AuthConfigs[registry] = newCreds

	file, err := json.MarshalIndent(auths, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("Error marshaling JSON: %s", err)
	}
	return file, err
}
