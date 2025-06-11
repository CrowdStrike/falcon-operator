package common

import "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"

type FalconCRD interface {
	*v1alpha1.FalconNodeSensor | *v1alpha1.FalconContainer | *v1alpha1.FalconAdmission | *v1alpha1.FalconImageAnalyzer

	GetFalconSecretSpec() v1alpha1.FalconSecret
	GetFalconAPISpec() *v1alpha1.FalconAPI
	SetFalconAPISpec(*v1alpha1.FalconAPI)
	GetFalconSpec() v1alpha1.FalconSensor
	SetFalconSpec(v1alpha1.FalconSensor)
}
