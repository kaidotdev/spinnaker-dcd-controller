module spinnaker-dcd-controller

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/mitchellh/cli v1.1.2 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/onsi/ginkgo v1.11.0 // indirect
	github.com/onsi/gomega v1.8.1 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spinnaker/roer v0.11.3
	github.com/spinnaker/spin v0.4.1-0.20201021165946-a6921971adf4
	github.com/urfave/cli v1.22.2 // indirect
	golang.org/x/xerrors v0.0.0-20191011141410-1b5146add898
	k8s.io/api v0.17.9
	k8s.io/apiextensions-apiserver v0.17.0 // indirect
	k8s.io/apimachinery v0.17.9
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/controller-tools v0.2.9 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.17.9
