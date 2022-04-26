package common

import (
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	sprigcrypto "github.com/crowdstrike/falcon-operator/pkg/sprig"
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

func FalconSensorConfig(falconsensor *falconv1alpha1.FalconSensor, cid string) map[string]string {
	sensorConfig := make(map[string]string)
	if cid != "" {
		sensorConfig["FALCONCTL_OPT_CID"] = cid
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

func FCAdmissionReviewVersions() []string {
	kubeVersion := GetKubernetesVersion()
	fcArv := []string{"v1"}

	if strings.Compare(kubeVersion.Minor, "22") < 0 {
		fcArv = []string{"v1", "v1beta"}
	}

	return fcArv
}

func GenCert(cn string, ips []interface{}, alternateDNS []interface{}, validity int, ca sprigcrypto.Certificate) sprigcrypto.Certificate {
	certs, err := sprigcrypto.GenerateSignedCertificate(cn, ips, alternateDNS, validity, ca)
	if err != nil {
		panic(err.Error())
	}

	return certs
}

func GenCA(cn string, validity int) sprigcrypto.Certificate {
	ca, err := sprigcrypto.GenerateCertificateAuthority(cn, validity)
	if err != nil {
		panic(err.Error())
	}
	return ca
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
