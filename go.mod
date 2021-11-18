module github.com/crowdstrike/falcon-operator

go 1.15

require (
	github.com/aws/aws-sdk-go-v2 v1.9.2
	github.com/aws/aws-sdk-go-v2/config v1.8.1
	github.com/aws/aws-sdk-go-v2/service/ecr v1.5.0
	github.com/containerd/containerd v1.5.8 // indirect
	github.com/containers/image/v5 v5.16.1
	github.com/crowdstrike/gofalcon v0.2.11
	github.com/fsnotify/fsnotify v1.5.0 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/openshift/api v0.0.0-20210428205234-a8389931bee7
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/sys v0.0.0-20210823070655-63515b42dcdf // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20210821163610-241b8fcbd6c8 // indirect
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v0.20.6
	sigs.k8s.io/controller-runtime v0.8.3
)
