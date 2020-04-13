# SpinnakerDCDController

SpinnakerDCDController is Kubernetes Custom Controller that manages [Spinnaker](https://github.com/spinnaker/spinnaker)'s Declarative Continuous Delivery resources.

## Installation

```shell
$ kubectl apply -k manifests
```

## Usage

Applying an `examples` manifest deploys Spinnaker resources.

```shell
$ kubectl apply -f examples
```

![spinnaker](https://github.com/kaidotdev/spinnaker-dcd-controller/wiki/images/spinnaker.png)

`spec` lives up to [dcd-spec](https://github.com/spinnaker/dcd-spec).

We use [roer](https://github.com/spinnaker/roer) internally that has become EOL, but we continue to use it because there is no alternative.
[spin](https://github.com/spinnaker/spin) is not a complete [roer](https://github.com/spinnaker/roer) successor.

## How to develop

### `skaffold dev`

```sh
$ make dev
```

### Test

```sh
$ make test
```

### Lint

```sh
$ make lint
```

### Generate CRD from `*_types.go` by controller-gen

```sh
$ make gen
```
