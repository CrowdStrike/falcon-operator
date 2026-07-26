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

// validateDefaultValues checks that CRD schema defaults are applied from kubebuilder:default annotations.
func (cr crConfig) validateDefaultValues() {
	By(fmt.Sprintf("validating default spec values for %s", cr.kind))

	if cr.kind != "FalconImageAnalyzer" {
		By("validating falcon.trace default is 'none'")
		cmd := exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.falcon.trace}")
		output, err := utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("none"),
			"spec.falcon.trace should default to 'none'")

		By("validating falcon.apd default is 'false'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.falcon.apd}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("false"),
			"spec.falcon.apd should default to false")
	}

	switch cr.kind {
	case "FalconNodeSensor":
		By("validating installNamespace default is 'falcon-system'")
		cmd := exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.installNamespace}")
		output, err := utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("falcon-system"),
			"spec.installNamespace should default to 'falcon-system'")

		By("validating node.backend default is 'bpf'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.node.backend}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("bpf"),
			"spec.node.backend should default to 'bpf'")

		By("validating node.imagePullPolicy default is 'Always'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.node.imagePullPolicy}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("Always"),
			"spec.node.imagePullPolicy should default to 'Always'")

		By("validating node.terminationGracePeriod default is 60")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.node.terminationGracePeriod}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("60"),
			"spec.node.terminationGracePeriod should default to 60")

		By("validating node.disableCleanup default is 'false'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.node.disableCleanup}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("false"),
			"spec.node.disableCleanup should default to false")

		By("validating node.updateStrategy.type default is 'RollingUpdate'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.node.updateStrategy.type}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("RollingUpdate"),
			"spec.node.updateStrategy.type should default to 'RollingUpdate'")

		By("validating node.tolerations include default control-plane tolerations")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.node.tolerations[*].key}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(ContainSubstring("node-role.kubernetes.io/master"),
			"spec.node.tolerations should include the master node-role toleration by default")
		ExpectWithOffset(1, string(output)).To(ContainSubstring("node-role.kubernetes.io/control-plane"),
			"spec.node.tolerations should include the control-plane node-role toleration by default")

	case "FalconContainer":
		By("validating installNamespace default is 'falcon-system'")
		cmd := exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.installNamespace}")
		output, err := utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("falcon-system"),
			"spec.installNamespace should default to 'falcon-system'")

		By("validating injector.listenPort default is 4433")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.injector.listenPort}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("4433"),
			"spec.injector.listenPort should default to 4433")

		By("validating injector.imagePullPolicy default is 'Always'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.injector.imagePullPolicy}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("Always"),
			"spec.injector.imagePullPolicy should default to 'Always'")

		By("validating injector.imagePullSecret default is 'crowdstrike-falcon-pull-secret'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.injector.imagePullSecret}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("crowdstrike-falcon-pull-secret"),
			"spec.injector.imagePullSecret should default to 'crowdstrike-falcon-pull-secret'")

		By("validating injector.replicas default is 2")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.injector.replicas}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("2"),
			"spec.injector.replicas should default to 2")

		By("validating injector.disableDefaultNamespaceInjection default is 'false'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.injector.disableDefaultNamespaceInjection}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("false"),
			"spec.injector.disableDefaultNamespaceInjection should default to false")

		By("validating injector.disableDefaultPodInjection default is 'false'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.injector.disableDefaultPodInjection}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("false"),
			"spec.injector.disableDefaultPodInjection should default to false")

	case "FalconAdmission":
		By("validating installNamespace default is 'falcon-kac'")
		cmd := exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.installNamespace}")
		output, err := utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("falcon-kac"),
			"spec.installNamespace should default to 'falcon-kac'")

		By("validating admissionConfig.servicePort default is 443")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.admissionConfig.servicePort}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("443"),
			"spec.admissionConfig.servicePort should default to 443")

		By("validating admissionConfig.containerPort default is 4443")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.admissionConfig.containerPort}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("4443"),
			"spec.admissionConfig.containerPort should default to 4443")

		By("validating admissionConfig.failurePolicy default is 'Ignore'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.admissionConfig.failurePolicy}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("Ignore"),
			"spec.admissionConfig.failurePolicy should default to 'Ignore'")

		By("validating admissionConfig.imagePullPolicy default is 'Always'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.admissionConfig.imagePullPolicy}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("Always"),
			"spec.admissionConfig.imagePullPolicy should default to 'Always'")

		By("validating admissionConfig.deployWatcher default is 'true'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.admissionConfig.deployWatcher}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("true"),
			"spec.admissionConfig.deployWatcher should default to true")

		By("validating admissionConfig.watcherEnabled default is 'true'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.admissionConfig.watcherEnabled}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("true"),
			"spec.admissionConfig.watcherEnabled should default to true")

		By("validating admissionConfig.snapshotsEnabled default is 'true'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.admissionConfig.snapshotsEnabled}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("true"),
			"spec.admissionConfig.snapshotsEnabled should default to true")

		By("validating admissionConfig.admissionControlEnabled default is 'true'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.admissionConfig.admissionControlEnabled}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("true"),
			"spec.admissionConfig.admissionControlEnabled should default to true")

		By("validating resourcequota.pods default is '2'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.resourcequota.pods}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("2"),
			"spec.resourcequota.pods should default to '2'")

	case "FalconImageAnalyzer":
		By("validating installNamespace default is 'falcon-iar'")
		cmd := exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.installNamespace}")
		output, err := utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("falcon-iar"),
			"spec.installNamespace should default to 'falcon-iar'")

		By("validating imageAnalyzerConfig.imagePullPolicy default is 'Always'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.imageAnalyzerConfig.imagePullPolicy}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("Always"),
			"spec.imageAnalyzerConfig.imagePullPolicy should default to 'Always'")

		By("validating imageAnalyzerConfig.sizeLimit default is '20Gi'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.imageAnalyzerConfig.sizeLimit}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("20Gi"),
			"spec.imageAnalyzerConfig.sizeLimit should default to '20Gi'")

		By("validating imageAnalyzerConfig.mountPath default is '/tmp'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.imageAnalyzerConfig.mountPath}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("/tmp"),
			"spec.imageAnalyzerConfig.mountPath should default to '/tmp'")

		By("validating imageAnalyzerConfig.registryConfig.autoDiscoverCredentials default is 'true'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.imageAnalyzerConfig.registryConfig.autoDiscoverCredentials}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("true"),
			"spec.imageAnalyzerConfig.registryConfig.autoDiscoverCredentials should default to true")

		By("validating imageAnalyzerConfig.iarAgentService.port default is 8001")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.imageAnalyzerConfig.iarAgentService.port}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("8001"),
			"spec.imageAnalyzerConfig.iarAgentService.port should default to 8001")

		By("validating imageAnalyzerConfig.kac.namespace default is 'falcon-kac'")
		cmd = exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", "jsonpath={.spec.imageAnalyzerConfig.kac.namespace}")
		output, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, string(output)).To(Equal("falcon-kac"),
			"spec.imageAnalyzerConfig.kac.namespace should default to 'falcon-kac'")
	}
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

// validateDefaultValues checks that CRD schema defaults are applied correctly
// when fields are not explicitly set in the manifest. These defaults are enforced
// by the Kubernetes API server from the kubebuilder:default annotations in the types.
func (cr crConfig) validateDefaultValues() {
	By(fmt.Sprintf("validating default spec values for %s", cr.kind))

	// Collect all validation failures instead of stopping at first failure
	var failures []string

	validateField := func(fieldPath, expectedValue, description string) {
		cmd := exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", fmt.Sprintf("jsonpath={%s}", fieldPath))
		output, err := utils.Run(cmd)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: failed to get value - %v", description, err))
			return
		}
		actualValue := string(output)
		if actualValue != expectedValue {
			failures = append(failures, fmt.Sprintf("%s: expected '%s', got '%s'", description, expectedValue, actualValue))
		}
	}

	validateContains := func(fieldPath, expectedSubstring, description string) {
		cmd := exec.Command("kubectl", "get", strings.ToLower(cr.kind), cr.metadataName,
			"-o", fmt.Sprintf("jsonpath={%s}", fieldPath))
		output, err := utils.Run(cmd)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: failed to get value - %v", description, err))
			return
		}
		actualValue := string(output)
		if !strings.Contains(actualValue, expectedSubstring) {
			failures = append(failures, fmt.Sprintf("%s: expected to contain '%s', got '%s'", description, expectedSubstring, actualValue))
		}
	}

	// Shared across all CRD kinds (except FalconImageAnalyzer which has no falcon sensor config)
	if cr.kind != "FalconImageAnalyzer" {
		validateField(".spec.falcon.trace", "none", "spec.falcon.trace")
		validateField(".spec.falcon.apd", "false", "spec.falcon.apd")
	}

	switch cr.kind {
	case "FalconNodeSensor":
		validateField(".spec.installNamespace", "falcon-system", "spec.installNamespace")
		validateField(".spec.node.backend", "bpf", "spec.node.backend")
		validateField(".spec.node.imagePullPolicy", "Always", "spec.node.imagePullPolicy")
		validateField(".spec.node.terminationGracePeriod", "60", "spec.node.terminationGracePeriod")
		validateField(".spec.node.disableCleanup", "false", "spec.node.disableCleanup")
		validateField(".spec.node.updateStrategy.type", "RollingUpdate", "spec.node.updateStrategy.type")
		validateContains(".spec.node.tolerations[*].key", "node-role.kubernetes.io/master", "spec.node.tolerations[master]")
		validateContains(".spec.node.tolerations[*].key", "node-role.kubernetes.io/control-plane", "spec.node.tolerations[control-plane]")

	case "FalconContainer":
		validateField(".spec.installNamespace", "falcon-system", "spec.installNamespace")
		validateField(".spec.injector.listenPort", "4433", "spec.injector.listenPort")
		validateField(".spec.injector.imagePullPolicy", "Always", "spec.injector.imagePullPolicy")
		validateField(".spec.injector.imagePullSecret", "crowdstrike-falcon-pull-secret", "spec.injector.imagePullSecret")
		validateField(".spec.injector.replicas", "2", "spec.injector.replicas")
		validateField(".spec.injector.disableDefaultNamespaceInjection", "false", "spec.injector.disableDefaultNamespaceInjection")
		validateField(".spec.injector.disableDefaultPodInjection", "false", "spec.injector.disableDefaultPodInjection")

	case "FalconAdmission":
		validateField(".spec.installNamespace", "falcon-kac", "spec.installNamespace")
		validateField(".spec.admissionConfig.servicePort", "443", "spec.admissionConfig.servicePort")
		validateField(".spec.admissionConfig.containerPort", "4443", "spec.admissionConfig.containerPort")
		validateField(".spec.admissionConfig.failurePolicy", "Ignore", "spec.admissionConfig.failurePolicy")
		validateField(".spec.admissionConfig.imagePullPolicy", "Always", "spec.admissionConfig.imagePullPolicy")
		validateField(".spec.admissionConfig.deployWatcher", "true", "spec.admissionConfig.deployWatcher")
		validateField(".spec.admissionConfig.watcherEnabled", "true", "spec.admissionConfig.watcherEnabled")
		validateField(".spec.admissionConfig.snapshotsEnabled", "true", "spec.admissionConfig.snapshotsEnabled")
		validateField(".spec.admissionConfig.admissionControlEnabled", "true", "spec.admissionConfig.admissionControlEnabled")
		validateField(".spec.admissionConfig.configMapWatcherEnabled", "true", "spec.admissionConfig.configMapWatcherEnabled")
		validateField(".spec.admissionConfig.snapshotsInterval", "22h", "spec.admissionConfig.snapshotsInterval")
		validateField(".spec.admissionConfig.falconImageAnalyzerNamespace", "falcon-iar", "spec.admissionConfig.falconImageAnalyzerNamespace")
		validateField(".spec.admissionConfig.replicas", "2", "spec.admissionConfig.replicas")
		validateField(".spec.resourcequota.pods", "2", "spec.resourcequota.pods")
		validateField(".spec.admissionConfig.updateStrategy.rollingUpdate.maxUnavailable", "0", "spec.admissionConfig.updateStrategy.rollingUpdate.maxUnavailable")
		validateField(".spec.admissionConfig.updateStrategy.rollingUpdate.maxSurge", "1", "spec.admissionConfig.updateStrategy.rollingUpdate.maxSurge")

	case "FalconImageAnalyzer":
		validateField(".spec.installNamespace", "falcon-iar", "spec.installNamespace")
		validateField(".spec.imageAnalyzerConfig.imagePullPolicy", "Always", "spec.imageAnalyzerConfig.imagePullPolicy")
		validateField(".spec.imageAnalyzerConfig.sizeLimit", "20Gi", "spec.imageAnalyzerConfig.sizeLimit")
		validateField(".spec.imageAnalyzerConfig.mountPath", "/tmp", "spec.imageAnalyzerConfig.mountPath")
		validateField(".spec.imageAnalyzerConfig.registryConfig.autoDiscoverCredentials", "true", "spec.imageAnalyzerConfig.registryConfig.autoDiscoverCredentials")
		validateField(".spec.imageAnalyzerConfig.iarAgentService.port", "8001", "spec.imageAnalyzerConfig.iarAgentService.port")
		validateField(".spec.imageAnalyzerConfig.kac.namespace", "falcon-kac", "spec.imageAnalyzerConfig.kac.namespace")
		validateField(".spec.imageAnalyzerConfig.updateStrategy.rollingUpdate.maxUnavailable", "0", "spec.imageAnalyzerConfig.updateStrategy.rollingUpdate.maxUnavailable")
		validateField(".spec.imageAnalyzerConfig.updateStrategy.rollingUpdate.maxSurge", "1", "spec.imageAnalyzerConfig.updateStrategy.rollingUpdate.maxSurge")

	case "FalconDeployment":
		// FalconDeployment level defaults
		validateField(".spec.registry.type", "crowdstrike", "spec.registry.type")
		validateField(".spec.deployAdmissionController", "true", "spec.deployAdmissionController")
		validateField(".spec.deployNodeSensor", "true", "spec.deployNodeSensor")
		validateField(".spec.deployImageAnalyzer", "true", "spec.deployImageAnalyzer")
		validateField(".spec.deployContainerSensor", "false", "spec.deployContainerSensor")

		// Nested FalconAdmission defaults (only if deployAdmissionController is true)
		validateField(".spec.falconAdmission.installNamespace", "falcon-kac", "spec.falconAdmission.installNamespace")
		validateField(".spec.falconAdmission.admissionConfig.servicePort", "443", "spec.falconAdmission.admissionConfig.servicePort")
		validateField(".spec.falconAdmission.admissionConfig.containerPort", "4443", "spec.falconAdmission.admissionConfig.containerPort")
		validateField(".spec.falconAdmission.admissionConfig.failurePolicy", "Ignore", "spec.falconAdmission.admissionConfig.failurePolicy")
		validateField(".spec.falconAdmission.admissionConfig.imagePullPolicy", "Always", "spec.falconAdmission.admissionConfig.imagePullPolicy")
		validateField(".spec.falconAdmission.admissionConfig.deployWatcher", "true", "spec.falconAdmission.admissionConfig.deployWatcher")
		validateField(".spec.falconAdmission.admissionConfig.watcherEnabled", "true", "spec.falconAdmission.admissionConfig.watcherEnabled")
		validateField(".spec.falconAdmission.admissionConfig.snapshotsEnabled", "true", "spec.falconAdmission.admissionConfig.snapshotsEnabled")
		validateField(".spec.falconAdmission.admissionConfig.snapshotsInterval", "22h", "spec.falconAdmission.admissionConfig.snapshotsInterval")
		validateField(".spec.falconAdmission.admissionConfig.admissionControlEnabled", "true", "spec.falconAdmission.admissionConfig.admissionControlEnabled")
		validateField(".spec.falconAdmission.admissionConfig.configMapWatcherEnabled", "true", "spec.falconAdmission.admissionConfig.configMapWatcherEnabled")
		validateField(".spec.falconAdmission.admissionConfig.falconImageAnalyzerNamespace", "falcon-iar", "spec.falconAdmission.admissionConfig.falconImageAnalyzerNamespace")
		validateField(".spec.falconAdmission.admissionConfig.replicas", "2", "spec.falconAdmission.admissionConfig.replicas")
		validateField(".spec.falconAdmission.resourcequota.pods", "2", "spec.falconAdmission.resourcequota.pods")
		validateField(".spec.falconAdmission.admissionConfig.updateStrategy.rollingUpdate.maxUnavailable", "0", "spec.falconAdmission.admissionConfig.updateStrategy.rollingUpdate.maxUnavailable")
		validateField(".spec.falconAdmission.admissionConfig.updateStrategy.rollingUpdate.maxSurge", "1", "spec.falconAdmission.admissionConfig.updateStrategy.rollingUpdate.maxSurge")

		// Nested FalconNodeSensor defaults (only if deployNodeSensor is true)
		validateField(".spec.falconNodeSensor.installNamespace", "falcon-system", "spec.falconNodeSensor.installNamespace")
		validateField(".spec.falconNodeSensor.node.backend", "bpf", "spec.falconNodeSensor.node.backend")
		validateField(".spec.falconNodeSensor.node.imagePullPolicy", "Always", "spec.falconNodeSensor.node.imagePullPolicy")
		validateField(".spec.falconNodeSensor.node.terminationGracePeriod", "60", "spec.falconNodeSensor.node.terminationGracePeriod")
		validateField(".spec.falconNodeSensor.node.disableCleanup", "false", "spec.falconNodeSensor.node.disableCleanup")
		validateField(".spec.falconNodeSensor.node.updateStrategy.type", "RollingUpdate", "spec.falconNodeSensor.node.updateStrategy.type")
		validateContains(".spec.falconNodeSensor.node.tolerations[*].key", "node-role.kubernetes.io/master", "spec.falconNodeSensor.node.tolerations[master]")
		validateContains(".spec.falconNodeSensor.node.tolerations[*].key", "node-role.kubernetes.io/control-plane", "spec.falconNodeSensor.node.tolerations[control-plane]")

		// Nested FalconImageAnalyzer defaults (only if deployImageAnalyzer is true)
		validateField(".spec.falconImageAnalyzer.installNamespace", "falcon-iar", "spec.falconImageAnalyzer.installNamespace")
		validateField(".spec.falconImageAnalyzer.imageAnalyzerConfig.imagePullPolicy", "Always", "spec.falconImageAnalyzer.imageAnalyzerConfig.imagePullPolicy")
		validateField(".spec.falconImageAnalyzer.imageAnalyzerConfig.sizeLimit", "20Gi", "spec.falconImageAnalyzer.imageAnalyzerConfig.sizeLimit")
		validateField(".spec.falconImageAnalyzer.imageAnalyzerConfig.mountPath", "/tmp", "spec.falconImageAnalyzer.imageAnalyzerConfig.mountPath")
		validateField(".spec.falconImageAnalyzer.imageAnalyzerConfig.registryConfig.autoDiscoverCredentials", "true", "spec.falconImageAnalyzer.imageAnalyzerConfig.registryConfig.autoDiscoverCredentials")
		validateField(".spec.falconImageAnalyzer.imageAnalyzerConfig.iarAgentService.port", "8001", "spec.falconImageAnalyzer.imageAnalyzerConfig.iarAgentService.port")
		validateField(".spec.falconImageAnalyzer.imageAnalyzerConfig.kac.namespace", "falcon-kac", "spec.falconImageAnalyzer.imageAnalyzerConfig.kac.namespace")
		validateField(".spec.falconImageAnalyzer.imageAnalyzerConfig.updateStrategy.rollingUpdate.maxUnavailable", "0", "spec.falconImageAnalyzer.imageAnalyzerConfig.updateStrategy.rollingUpdate.maxUnavailable")
		validateField(".spec.falconImageAnalyzer.imageAnalyzerConfig.updateStrategy.rollingUpdate.maxSurge", "1", "spec.falconImageAnalyzer.imageAnalyzerConfig.updateStrategy.rollingUpdate.maxSurge")

		// Nested FalconContainerSensor defaults (only if deployContainerSensor is true)
		validateField(".spec.falconContainerSensor.installNamespace", "falcon-system", "spec.falconContainerSensor.installNamespace")
		validateField(".spec.falconContainerSensor.injector.listenPort", "4433", "spec.falconContainerSensor.injector.listenPort")
		validateField(".spec.falconContainerSensor.injector.imagePullPolicy", "Always", "spec.falconContainerSensor.injector.imagePullPolicy")
		validateField(".spec.falconContainerSensor.injector.imagePullSecret", "crowdstrike-falcon-pull-secret", "spec.falconContainerSensor.injector.imagePullSecret")
		validateField(".spec.falconContainerSensor.injector.replicas", "2", "spec.falconContainerSensor.injector.replicas")
		validateField(".spec.falconContainerSensor.injector.disableDefaultNamespaceInjection", "false", "spec.falconContainerSensor.injector.disableDefaultNamespaceInjection")
		validateField(".spec.falconContainerSensor.injector.disableDefaultPodInjection", "false", "spec.falconContainerSensor.injector.disableDefaultPodInjection")

		// Shared falcon sensor config defaults for nested specs
		validateField(".spec.falconAdmission.falcon.trace", "none", "spec.falconAdmission.falcon.trace")
		validateField(".spec.falconAdmission.falcon.apd", "false", "spec.falconAdmission.falcon.apd")
		validateField(".spec.falconNodeSensor.falcon.trace", "none", "spec.falconNodeSensor.falcon.trace")
		validateField(".spec.falconNodeSensor.falcon.apd", "false", "spec.falconNodeSensor.falcon.apd")
		validateField(".spec.falconContainerSensor.falcon.trace", "none", "spec.falconContainerSensor.falcon.trace")
		validateField(".spec.falconContainerSensor.falcon.apd", "false", "spec.falconContainerSensor.falcon.apd")
	}

	// Report all failures at once
	if len(failures) > 0 {
		failureMsg := fmt.Sprintf("\n❌ Default value validation failed for %s:\n", cr.kind)
		for _, failure := range failures {
			failureMsg += fmt.Sprintf("  • %s\n", failure)
		}
		fmt.Fprint(GinkgoWriter, failureMsg)
		Fail(failureMsg)
	} else {
		By(fmt.Sprintf("✓ All %s default values validated successfully", cr.kind))
	}
}
