module spinnaker-dcd-controller

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/google/uuid v1.1.1
	github.com/mitchellh/mapstructure v1.1.2
	github.com/spinnaker/roer v0.11.3
	github.com/urfave/cli v1.22.2 // indirect
	golang.org/x/tools v0.0.0-20191227053925-7b8e75db28f4 // indirect
	k8s.io/api v0.0.0-20190918155943-95b840bb6a1f
	k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90
	sigs.k8s.io/controller-runtime v0.4.0
)
