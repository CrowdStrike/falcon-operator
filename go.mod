module github.com/crowdstrike/falcon-operator

go 1.15

require (
	github.com/containers/image/v5 v5.15.2
	github.com/crowdstrike/gofalcon v0.2.7
	github.com/go-logr/logr v0.3.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.3
	github.com/openshift/api v0.0.0-20201120165435-072a4cd8ca42
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v0.20.6
	sigs.k8s.io/controller-runtime v0.7.0
)
