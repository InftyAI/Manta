<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/inftyai/manta/main/docs/assets/logo.png">
    <img alt="manta" src="https://raw.githubusercontent.com/inftyai/manta/main/docs/assets/logo.png" width="35%">
  </picture>
</p>

<h3 align="center">
A lightweight P2P-based cache system for model distributions on Kubernetes.
</h3>

[![stability-alpha](https://img.shields.io/badge/stability-alpha-f4d03f.svg)](https://github.com/mkenney/software-guides/blob/master/STABILITY-BADGES.md#alpha)
[![GoReport Widget]][GoReport Status]
[![Latest Release](https://img.shields.io/github/v/release/inftyai/manta?include_prereleases)](https://github.com/inftyai/manta/releases/latest)

[GoReport Widget]: https://goreportcard.com/badge/github.com/inftyai/manta
[GoReport Status]: https://goreportcard.com/report/github.com/inftyai/manta

_Name Story: the inspiration of the name `Manta` is coming from Dota2, called [Manta Style](https://dota2.fandom.com/wiki/Manta_Style), which will create 2 images of your hero just like peers in the P2P network._


## Architecture

![architecture](./docs/assets/arch.png)

> Note: [llmaz](https://github.com/InftyAI/llmaz) is just one kind of integrations, **Manta** can be deployed and used independently.

## Features Overview

- **Model Hub Support**: Models could be downloaded directly from model hubs (Huggingface etc.) or object storages, no other efforts.
- **Model Preheat**: Models could be preloaded to clusters, or specified nodes to accelerate the model serving.
- **Model Cache**: Models will be cached as chunks after downloading for faster model loading.
- **Model Lifecycle Management**: Model lifecycle is managed automatically with different strategies, like `Retain` or `Delete`.
- **Plugin Framework**: _Filter_ and _Score_ plugins could be extended to pick up the best candidates.
- **Memory Management(WIP)**: Manage the reserved memories for caching, together with LRU algorithm for GC.

## You Should Know Before

- Manta is not an all-in-one solution for model management, instead, it offers a lightweight solution to utilize the idle bandwidth and cost-effective disk, helping you save money.
- It requires no additional components like databases or storage systems, simplifying setup and reducing effort.
- All the models will be stored under the host path of `/mnt/models/`
- After all, it's just a **cache system**.

## Quick Start

### Installation

Read the [Installation](./docs//installation.md) for guidance.

### Preheat Models

A sample to preload the `Qwen/Qwen2.5-0.5B-Instruct` model:

```yaml
apiVersion: manta.io/v1alpha1
kind: Torrent
metadata:
  name: torrent-sample
spec:
  hub:
    name: Huggingface
    repoID: Qwen/Qwen2.5-0.5B-Instruct
```

If you want to preload the model to specified nodes, use the `NodeSelector`:

```yaml
apiVersion: manta.io/v1alpha1
kind: Torrent
metadata:
  name: torrent-sample
spec:
  hub:
    name: Huggingface
    repoID: Qwen/Qwen2.5-0.5B-Instruct
  nodeSelector:
    zone: zone-a
```

### Delete Models

If you want to remove the model weights once `Torrent` is deleted, set the `ReclaimPolicy=Delete`, default to `Retain`:

```yaml
apiVersion: manta.io/v1alpha1
kind: Torrent
metadata:
  name: torrent-sample
spec:
  hub:
    name: Huggingface
    repoID: Qwen/Qwen2.5-0.5B-Instruct
  reclaimPolicy: Delete
```

More details refer to the [APIs](https://github.com/InftyAI/Manta/blob/main/api/v1alpha1/torrent_types.go).

## Roadmap

- Preloading datasets from model hubs
- RDMA support for faster model loading
- More integrations with serving projects

## Community

Join us for more discussions:

* **Slack Channel**: [#manta](https://inftyai.slack.com/archives/C07SY8WS45U)

## Contributions

All kinds of contributions are welcomed ! Please following [CONTRIBUTING.md](./CONTRIBUTING.md).
