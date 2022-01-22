package common

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	sprigcrypto "github.com/crowdstrike/falcon-operator/pkg/sprig"
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
	m := make(map[string]string)
	var cmOptInt map[string]interface{}
	jsonCmOpt, err := json.Marshal(falconContainer)
	if err != nil {
		return nil
	}

	err = json.Unmarshal(jsonCmOpt, &cmOptInt)
	if err != nil {
		return nil
	}

	// iterate through jsonCmOpt
	for field, val := range cmOptInt {
		if field != "" {
			// Make the keys match the env variable names for now
			key := "FALCONCTL_OPT_" + strings.ToUpper(field)

			switch v := val.(type) {
			case bool:
				m[key] = strconv.FormatBool(v)
			case string:
				m[key] = v
			default:
				return m
			}
		}
	}

	return m
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

func DecodedBase64(s string) string {
	b64byte, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`[\t|\n]*`)
	cleanup := re.ReplaceAllString(string(b64byte), "")

	return cleanup
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
	if cid != nil {
		return *cid, nil
	}

	client, err := falcon.NewClient(fa)
	if err != nil {
		return "", err
	}
	return falcon_api.CCID(ctx, client)
}
