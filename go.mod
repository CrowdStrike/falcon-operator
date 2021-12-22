module github.com/crowdstrike/falcon-operator

go 1.15

require (
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/Microsoft/hcsshim v0.9.1 // indirect
	github.com/aws/aws-sdk-go-v2 v1.11.2
	github.com/aws/aws-sdk-go-v2/config v1.11.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.11.1
	github.com/containerd/cgroups v1.0.2 // indirect
	github.com/containerd/containerd v1.5.8 // indirect
	github.com/containers/image/v5 v5.17.0
	github.com/crowdstrike/gofalcon v0.2.16
	github.com/fsnotify/fsnotify v1.5.0 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/moby/sys/mountinfo v0.5.0 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/opencontainers/selinux v1.10.0 // indirect
	github.com/openshift/api v0.0.0-20210428205234-a8389931bee7
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/net v0.0.0-20211123203042-d83791d6bcd9 // indirect
	golang.org/x/sys v0.0.0-20211123173158-ef496fb156ab // indirect
	golang.org/x/tools v0.1.5 // indirect
	google.golang.org/genproto v0.0.0-20211118181313-81c1377c94b1 // indirect
	google.golang.org/grpc v1.42.0 // indirect
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v0.20.6
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.2-0.20210819154149-5ad6f50d6283
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.3
)
