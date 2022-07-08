package common

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	"github.com/crowdstrike/gofalcon/falcon"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func InitContainerArgs() []string {
	return []string{
		"-c",
		"mkdir -p " + FalconDataDir +
			" && " +
			"touch " + FalconStoreFile,
	}
}

func GetFalconImage(nodesensor *falconv1alpha1.FalconNodeSensor) string {
	if nodesensor.Spec.Node.Image == "" {
		return FalconDefaultImage
	}
	return nodesensor.Spec.Node.Image
}

func FalconSensorConfig(falconsensor *falconv1alpha1.FalconSensor) map[string]string {
	sensorConfig := make(map[string]string)
	if falconsensor.CID != "" {
		sensorConfig["FALCONCTL_OPT_CID"] = falconsensor.CID
	}
	if falconsensor.APD != nil {
		sensorConfig["FALCONCTL_OPT_APD"] = strconv.FormatBool(*falconsensor.APD)
	}
	if falconsensor.APH != "" {
		sensorConfig["FALCONCTL_OPT_APH"] = falconsensor.APH
	}
	if falconsensor.APP != nil {
		sensorConfig["FALCONCTL_OPT_APP"] = strconv.Itoa(*falconsensor.APP)
	}
	if falconsensor.Billing != "" {
		sensorConfig["FALCONCTL_OPT_BILLING"] = falconsensor.Billing
	}
	if falconsensor.PToken != "" {
		sensorConfig["FALCONCTL_OPT_PROVISIONING_TOKEN"] = falconsensor.PToken
	}
	if len(falconsensor.Tags) > 0 {
		sensorConfig["FALCONCTL_OPT_TAGS"] = strings.Join(falconsensor.Tags, ",")
	}
	if falconsensor.Trace != "" {
		sensorConfig["FALCONCTL_OPT_TRACE"] = falconsensor.Trace
	}

	return sensorConfig
}

func FalconContainerConfig(falconContainer *falconv1alpha1.FalconContainer) map[string]string {
	sensorConfig := make(map[string]string)
	if falconContainer.Spec.FalconContainerSensor.CID != "" {
		sensorConfig["FALCONCTL_OPT_CID"] = falconContainer.Spec.FalconContainerSensor.CID
	}
	if falconContainer.Spec.FalconContainerSensor.APD != nil {
		sensorConfig["FALCONCTL_OPT_APD"] = strconv.FormatBool(*falconContainer.Spec.FalconContainerSensor.APD)
	}
	if falconContainer.Spec.FalconContainerSensor.APH != "" {
		sensorConfig["FALCONCTL_OPT_APH"] = falconContainer.Spec.FalconContainerSensor.APH
	}
	if falconContainer.Spec.FalconContainerSensor.APP != nil {
		sensorConfig["FALCONCTL_OPT_APP"] = strconv.Itoa(*falconContainer.Spec.FalconContainerSensor.APP)
	}
	if falconContainer.Spec.FalconContainerSensor.Billing != "" {
		sensorConfig["FALCONCTL_OPT_BILLING"] = falconContainer.Spec.FalconContainerSensor.Billing
	}
	if falconContainer.Spec.FalconContainerSensor.PToken != "" {
		sensorConfig["FALCONCTL_OPT_PROVISIONING_TOKEN"] = falconContainer.Spec.FalconContainerSensor.PToken
	}
	if len(falconContainer.Spec.FalconContainerSensor.Tags) > 0 {
		sensorConfig["FALCONCTL_OPT_TAGS"] = strings.Join(falconContainer.Spec.FalconContainerSensor.Tags, ",")
	}
	if falconContainer.Spec.FalconContainerSensor.Trace != "" {
		sensorConfig["FALCONCTL_OPT_TRACE"] = falconContainer.Spec.FalconContainerSensor.Trace
	}

	sensorConfig["CP_NAMESPACE"] = falconContainer.Namespace
	sensorConfig["FALCON_IMAGE"] = falconContainer.Spec.FalconContainerSensorConfig.Image
	sensorConfig["FALCON_IMAGE_PULL_POLICY"] = string(falconContainer.Spec.FalconContainerSensorConfig.ImagePullPolicy)
	sensorConfig["FALCON_INJECTOR_LISTEN_PORT"] = fmt.Sprintf("%d", falconContainer.Spec.FalconContainerSensorConfig.InjectorPort)

	resources, _ := json.Marshal(falconContainer.Spec.FalconContainerSensor.ContainerResources)
	if string(resources) != "null" {
		sensorConfig["FALCON_RESOURCES"] = string(EncodedBase64String(string(resources)))
	}
	if falconContainer.Spec.FalconContainerSensorConfig.DisablePodInjection {
		sensorConfig["INJECTION_DEFAULT_DISABLED"] = "T"
	}
	if len(falconContainer.Spec.FalconContainerSensorConfig.ContainerDaemonSocket) > 0 {
		sensorConfig["SENSOR_CTR_RUNTIME_SOCKET_PATH"] = falconContainer.Spec.FalconContainerSensorConfig.ContainerDaemonSocket
	}

	return sensorConfig
}

func FCAdmissionReviewVersions() []string {
	kubeVersion := GetKubernetesVersion()
	fcArv := []string{"v1"}

	if strings.Compare(kubeVersion.Minor, "22") < 0 {
		fcArv = []string{"v1", "v1beta"}
	}

	return fcArv
}

func AltDNSListGenerator(name string, namespace string) []string {
	list := []string{}

	for _, suffix := range altDNSSuffixList {
		list = append(list, fmt.Sprintf("%s.%s.%s", name, namespace, suffix))
	}

	list = append(list, fmt.Sprintf("%s.%s", name, namespace))
	list = append(list, name)
	return list
}

func GenCerts(cn string, name string, ips []string, alternateDNS []string, validity int) (CAcert, Certs) {
	var buf bytes.Buffer
	var altDNS []string
	var ip []string
	ca := CAcert{}
	certs := Certs{}

	if len(ips) > 0 {
		for _, suffix := range ips {
			ip = append(ip, fmt.Sprintf("\"%s\"", suffix))
		}
	}

	if len(alternateDNS) > 0 {
		for _, suffix := range alternateDNS {
			altDNS = append(altDNS, fmt.Sprintf("\"%s\"", suffix))

		}
	}

	tpl := fmt.Sprintf(`{{- $ca := genCA "%s" %d -}}{{- $cert := genSignedCert "%s" (list %s) (list %s) %d $ca -}}{{ $ca }},{{ $cert}}`, cn, validity, name, strings.Join(ip, " "), strings.Join(altDNS, " "), validity)

	t := template.Must(template.New("test").Funcs(sprig.TxtFuncMap()).Parse(tpl))
	if err := t.Execute(&buf, nil); err != nil {
		fmt.Printf("Error during template execution: %s", err)
		return ca, certs
	}
	strip := regexp.MustCompile(`[{|}]+`)
	re := regexp.MustCompile(`(\n )+`)

	fullCA := strings.Split(buf.String(), ",")[0]

	v := re.Split(strip.ReplaceAllString(fullCA, ""), -1)
	ca.Cert = v[0]
	ca.Key = v[1]

	sysCerts := strings.Split(buf.String(), ",")[1]
	v = re.Split(strip.ReplaceAllString(sysCerts, ""), -1)
	certs.Cert = v[0]
	certs.Key = v[1]

	return ca, certs
}

func EncodedBase64String(data string) []byte {
	base64EncodedData := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(base64EncodedData, []byte(data))
	return base64EncodedData
}

func CleanDecodedBase64(s []byte) []byte {
	re := regexp.MustCompile(`[\t|\n]*`)
	b64byte, err := base64.StdEncoding.DecodeString(string(s))
	if err != nil {
		return []byte(re.ReplaceAllString(string(s), ""))
	}
	return []byte(re.ReplaceAllString(string(b64byte), ""))
}

func GetKubernetesVersion() *version.Info {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	version, err := clientset.DiscoveryClient.ServerVersion()
	if err != nil {
		panic(err.Error())
	}

	return version
}

func FalconCID(ctx context.Context, cid *string, fa *falcon.ApiConfig) (string, error) {
	fa.Context = ctx
	if cid != nil {
		return *cid, nil
	}

	client, err := falcon.NewClient(fa)
	if err != nil {
		return "", err
	}
	return falcon_api.CCID(ctx, client)
}
