# Installation Guide

## Prerequisites

* Kubernetes version >= 1.18

## Install a released version

### Install

```cmd
VERSION=v0.0.1
kubectl apply --server-side -f https://github.com/inftyai/manta/releases/download/$VERSION/manifests.yaml
```

### Uninstall

```cmd
VERSION=v0.0.1
kubectl delete -f https://github.com/inftyai/manta/releases/download/$VERSION/manifests.yaml
```

## Install from source

### Install

```cmd
git clone https://github.com/inftyai/manta.git
cd manta
IMG=<IMAGE_REGISTRY>/<IMAGE_NAME>:<GIT_TAG> make image-push deploy
```

### Uninstall

```cmd
make undeploy
```
