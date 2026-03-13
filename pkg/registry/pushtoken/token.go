package pushtoken

import (
	"context"
	"fmt"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/aws"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"github.com/crowdstrike/falcon-operator/pkg/registry/auth"
	"github.com/go-logr/logr"
)

// GetCredentials returns container pushtoken authentication information that can be used to authenticate container push requests.
func GetCredentials(ctx context.Context, log logr.Logger, registryType falconv1alpha1.RegistryTypeSpec, query k8s_utils.KubeQuerySecretsMethod) (auth.Credentials, error) {
	log.Info("Getting push credentials", "registryType", registryType)

	switch registryType {
	case falconv1alpha1.RegistryTypeECR:
		log.Info("Using ECR authentication")
		cfg, err := aws.NewConfig()
		if err != nil {
			log.Error(err, "Failed to create AWS config")
			return nil, err
		}
		token, err := cfg.ECRLogin(ctx)
		if err != nil {
			log.Error(err, "Failed to get ECR login token")
			return nil, err
		}
		log.Info("Successfully obtained ECR credentials")
		return auth.ECRCredentials(string(token))
	default:
		log.Info("Querying secrets for push credentials")
		secrets, err := query(ctx)
		if err != nil {
			log.Error(err, "Failed to query secrets")
			return nil, err
		}

		log.Info("Query returned secrets", "secretCount", len(secrets.Items))

		// Log details about each secret found
		for i, secret := range secrets.Items {
			annotations := ""
			if secret.ObjectMeta.Annotations != nil {
				if saName, ok := secret.ObjectMeta.Annotations["kubernetes.io/service-account.name"]; ok {
					annotations = fmt.Sprintf("service-account.name=%s", saName)
				}
			}
			log.Info("Secret found",
				"index", i,
				"name", secret.Name,
				"namespace", secret.Namespace,
				"type", secret.Type,
				"annotations", annotations,
				"hasData", secret.Data != nil,
			)
		}

		creds := auth.GetPushCredentials(log, secrets.Items)
		if creds == nil {
			log.Error(nil, "No suitable secret found for push credentials",
				"requiredSecretName", "builder",
				"requiredAnnotation", "kubernetes.io/service-account.name=builder",
				"requiredTypes", "kubernetes.io/dockercfg OR kubernetes.io/dockerconfigjson",
				"secretsChecked", len(secrets.Items),
			)
			return nil, fmt.Errorf("Cannot find suitable secret to push falcon-image to your registry")
		}
		log.Info("Successfully found push credentials", "secretName", creds.Name())
		return creds, nil
	}
}
