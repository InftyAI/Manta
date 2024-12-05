# Installation Guide

## Prerequisites

* Kubernetes version >= 1.18

## Install a released version

### Install

```cmd
VERSION=v0.0.4
kubectl apply --server-side -f https://github.com/inftyai/manta/releases/download/$VERSION/manifests.yaml
```

After installation, you will see outputs like：

- ` kubectl get pods -n manta-system`
    ```
    NAME                                        READY   STATUS    RESTARTS   AGE
    manta-agent-jdnt5                           1/1     Running   0          77s
    manta-agent-n55rq                           1/1     Running   0          77s
    manta-controller-manager-567b565c54-pzvlc   2/2     Running   0          77s
    ```
- `kubectl get nodetrackers`
    ```
    NAME           AGE
    kind-worker    35s
    kind-worker2   29s
    ```

### Uninstall

```cmd
VERSION=v0.0.4
kubectl delete -f https://github.com/inftyai/manta/releases/download/$VERSION/manifests.yaml
```

## Install from source

### Install

```cmd
git clone https://github.com/inftyai/manta.git
cd manta
IMG=<IMAGE_REGISTRY>/<IMAGE_NAME>:<GIT_TAG> make image-push deploy
```

After installation, you will see outputs like：

- ` kubectl get pods -n manta-system`
    ```
    NAME                                        READY   STATUS    RESTARTS   AGE
    manta-agent-jdnt5                           1/1     Running   0          77s
    manta-agent-n55rq                           1/1     Running   0          77s
    manta-controller-manager-567b565c54-pzvlc   2/2     Running   0          77s
    ```
- `kubectl get nodetrackers`
    ```
    NAME           AGE
    kind-worker    35s
    kind-worker2   29s
    ```

### Uninstall

```cmd
make undeploy
```
