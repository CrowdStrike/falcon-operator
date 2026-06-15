package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/crowdstrike/falcon-operator/test/utils"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"
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

// createFalconSecret creates the falcon-secrets namespace and k8s secret with Falcon API credentials
func createFalconSecret() {
	falconClientID, falconClientSecret := getCredentials()

	By("creating a k8s secret with Falcon API credentials")
	if falconClientID != "" && falconClientSecret != "" {
		createNamespaceCmd := exec.Command("kubectl", "create", "ns", falconSecretNamespace)
		_, err := utils.Run(createNamespaceCmd)
		ExpectWithOffset(2, err).NotTo(HaveOccurred())

		createSecretCmd := exec.Command("sh", "-c",
			fmt.Sprintf("kubectl create secret generic %s -n %s --from-literal=falcon-client-id=\"$FALCON_CLIENT_ID\" --from-literal=falcon-client-secret=\"$FALCON_CLIENT_SECRET\"",
				falconSecretName, falconSecretNamespace))
		_, err = utils.Run(createSecretCmd)
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
	}
}

func getAITapCredentials() (aidrToken string, aidrBaseUrl string) {
	aidrToken = os.Getenv("FALCON_AIDR_TOKEN")
	if aidrToken == "" {
		aidrToken = "test-aidr-token-placeholder"
		By("FALCON_AIDR_TOKEN not set, using placeholder token for testing")
	}

	aidrBaseUrl = os.Getenv("FALCON_AIDR_BASE_URL")
	if aidrBaseUrl == "" {
		aidrBaseUrl = "https://test.aidr-base-url.com"
		By("FALCON_AIDR_BASE_URL not set, using placeholder base URL for testing")
	}
	return aidrToken, aidrBaseUrl
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

// loadManifest reads a manifest file, substitutes Falcon API credential placeholders
// and any extra substitutions in the raw YAML, and returns the resulting bytes.
func loadManifest(manifest string, extra ...map[string]string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, manifest))
	if err != nil {
		return nil, err
	}

	content := string(data)

	falconClientID, falconClientSecret := getCredentials()
	if falconClientID != "" && falconClientSecret != "" {
		content = strings.ReplaceAll(content, "client_id: PLEASE_FILL_IN", fmt.Sprintf("client_id: %s", falconClientID))
		content = strings.ReplaceAll(content, "client_secret: PLEASE_FILL_IN", fmt.Sprintf("client_secret: %s", falconClientSecret))
	}

	for _, replacements := range extra {
		for old, new := range replacements {
			content = strings.ReplaceAll(content, old, new)
		}
	}

	return []byte(content), nil
}

// applyManifest applies raw YAML bytes via kubectl with an optional namespace.
func applyManifest(data []byte, namespace string) error {
	args := []string{"apply", "-f", "-"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	cmd := exec.Command("kubectl", args...)
	cmd.Stdin = strings.NewReader(string(data))
	_, err := utils.Run(cmd)
	return err
}
