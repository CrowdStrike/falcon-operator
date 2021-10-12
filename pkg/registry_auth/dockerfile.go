package registry_auth

import (
	"encoding/json"
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

