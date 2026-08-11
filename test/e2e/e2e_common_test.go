package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/test/utils"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"
)

type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}

func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() string {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	metricsOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return string(metricsOutput)
}

func getCredentials() (client_id string, client_secret string) {
	if clientID, ok := os.LookupEnv("FALCON_CLIENT_ID"); ok {
		client_id = clientID
	}

	if clientSecret, ok := os.LookupEnv("FALCON_CLIENT_SECRET"); ok {
		client_secret = clientSecret
	}
	return client_id, client_secret
}

func updateManifestApiCreds(manifest string) {
	falconClientID, falconClientSecret := getCredentials()

	if falconClientID != "" && falconClientSecret != "" {
		err := utils.ReplaceInFile(
			filepath.Join(projectDir, manifest),
			"client_id: PLEASE_FILL_IN", fmt.Sprintf("client_id: %s", falconClientID))
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		err = utils.ReplaceInFile(filepath.Join(projectDir, manifest),
			"client_secret: PLEASE_FILL_IN", fmt.Sprintf("client_secret: %s", falconClientSecret))
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
	}
}

func addFalconSecretToManifest(manifest string) {
	falconClientID, falconClientSecret := getCredentials()

	By("creating a k8s secret with Falcon API credentials")
	if falconClientID != "" && falconClientSecret != "" {
		// Create secret namespace and secret
		createNamespaceCmd := exec.Command("kubectl", "create", "ns", falconSecretNamespace)
		_, err := utils.Run(createNamespaceCmd)
		ExpectWithOffset(2, err).NotTo(HaveOccurred())

		createSecretCmd := exec.Command("sh", "-c",
			fmt.Sprintf("kubectl create secret generic %s -n %s --from-literal=falcon-client-id=\"$FALCON_CLIENT_ID\" --from-literal=falcon-client-secret=\"$FALCON_CLIENT_SECRET\"",
				falconSecretName, falconSecretNamespace))
		_, err = utils.Run(createSecretCmd)
		ExpectWithOffset(2, err).NotTo(HaveOccurred())

		err = utils.ReplaceInFile(filepath.Join(projectDir, manifest),
			"namespace: PLEASE_FILL_IN", fmt.Sprintf("namespace: %s", falconSecretNamespace))
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		err = utils.ReplaceInFile(filepath.Join(projectDir, manifest),
			"secretName: PLEASE_FILL_IN", fmt.Sprintf("secretName: %s", falconSecretName))
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
	}
}

func updateManifestWithAITapToken(manifest string) {
	aidrToken := os.Getenv("FALCON_AIDR_TOKEN")
	if aidrToken == "" {
		aidrToken = "test-aidr-token-placeholder"
		By("FALCON_AIDR_TOKEN not set, using placeholder token for testing")
	}

	By("updating manifest with AITap AI-DR token")
	err := utils.ReplaceInFile(filepath.Join(projectDir, manifest),
		"aidrCollectorApiToken: PLEASE_FILL_IN", fmt.Sprintf("aidrCollectorApiToken: %s", aidrToken))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
}

func updateManifestWithAITapBaseURL(manifest string) {
	aidrBaseUrl := os.Getenv("FALCON_AIDR_BASE_URL")
	if aidrBaseUrl == "" {
		aidrBaseUrl = "https://test.aidr-base-url.com"
		By("FALCON_AIDR_BASE_URL not set, using placeholder base URL for testing")
	}

	By("updating manifest with AITap AI-DR base URL")
	err := utils.ReplaceInFile(filepath.Join(projectDir, manifest),
		"aidrCollectorBaseApiUrl: PLEASE_FILL_IN", fmt.Sprintf("aidrCollectorBaseApiUrl: %s", aidrBaseUrl))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
}

func validateAITapSecrets() {
	By("validating AITap AI-DR secret exists in default namespace")
	cmd := exec.Command("kubectl", "get", "secret", "falcon-aitap-aidr-secret", "-n", "default")
	output, err := utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, string(output)).To(ContainSubstring("falcon-aitap-aidr-secret"))

	By("validating AITap AI-DR secret contains .collector-aidr-token key")
	cmd = exec.Command("kubectl", "get", "secret", "falcon-aitap-aidr-secret", "-n", "default", "-o", "jsonpath={.data.\\.collector-aidr-token}")
	output, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, len(output)).To(BeNumerically(">", 0))

	By("validating ConfigMap contains AITap environment variables")
	cmd = exec.Command("kubectl", "get", "configmap", "falcon-sidecar-injector-config", "-n", "falcon-system", "-o", "jsonpath={.data.FALCON_AITAP_AIDR_SECRET_NAME}")
	output, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, string(output)).To(Equal("falcon-aitap-aidr-secret"))
}

func (c crConfig) validateInitContainerReadOnlyRootFilesystem() {
	By("validating that the init container has readOnlyRootFilesystem set to true")
	cmd := exec.Command("kubectl", "get", "daemonset", "falcon-node-sensor", "-n", c.namespace, "-o", "jsonpath={.spec.template.spec.initContainers[0].securityContext.readOnlyRootFilesystem}")
	output, err := utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, string(output)).To(Equal("true"))
}

// loadManifest reads a manifest file, unmarshals it into the provided object, and updates credentials
func loadManifest(manifest string, obj any) error {
	manifestPath := filepath.Join(projectDir, manifest)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, obj); err != nil {
		return err
	}

	falconClientID, falconClientSecret := getCredentials()
	if falconClientID == "" || falconClientSecret == "" {
		return nil
	}

	switch v := obj.(type) {
	case *falconv1alpha1.FalconImageAnalyzer:
		v.Spec.FalconAPI.ClientId = falconClientID
		v.Spec.FalconAPI.ClientSecret = falconClientSecret
	case *falconv1alpha1.FalconAdmission:
		v.Spec.FalconAPI.ClientId = falconClientID
		v.Spec.FalconAPI.ClientSecret = falconClientSecret
	case *falconv1alpha1.FalconNodeSensor:
		v.Spec.FalconAPI.ClientId = falconClientID
		v.Spec.FalconAPI.ClientSecret = falconClientSecret
	case *falconv1alpha1.FalconContainer:
		v.Spec.FalconAPI.ClientId = falconClientID
		v.Spec.FalconAPI.ClientSecret = falconClientSecret
	case *falconv1alpha1.FalconDeployment:
		v.Spec.FalconAPI.ClientId = falconClientID
		v.Spec.FalconAPI.ClientSecret = falconClientSecret
	}

	return nil
}

// applyManifest marshals the object to YAML and applies it via kubectl
func applyManifest(obj any, namespace string) error {
	// Marshal to YAML
	data, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	// Apply via kubectl
	cmd := exec.Command("kubectl", "apply", "-f", "-", "-n", namespace)
	cmd.Stdin = strings.NewReader(string(data))
	_, err = utils.Run(cmd)
	return err
}
