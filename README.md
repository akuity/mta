# mta

The `mta` cli will export Flux components to Argo CD consumable
CRs. This can be used in order to help migrating from Flux to Argo
CD. This is in the "proof of concept" phase and I make no guarantees

Currently working:

- [x] Migrate Kustomizations
- [x] Migrate HelmReleases
- [x] Scan
- [ ] Auto Migrate
- [ ] Uninstall Flux

# Installation

Install the `mta` binary from the releases page

__Linux/Mac OS X86_64__

```shell
sudo wget -O /usr/local/bin/mta https://github.com/christianh814/mta/releases/download/v0.0.3/mta-amd64-$(uname -s | tr [:upper:] [:lower:])
```

__Mac OS Apple Silicon__

```shell
sudo wget -O /usr/local/bin/mta https://github.com/christianh814/mta/releases/download/v0.0.3/mta-arm64-darwin
```

Make sure it's executable

```shell
sudo chmod +x /usr/local/bin/mta
```

There is bash completion

> *NOTE* it's probably `zsh` on a Mac

```shell
mta completion bash
```

# Quickstarts

Quickstarts to test the functionality after downloading the CLI

* [Kustomizations](docs/kustomizations.md)
* [HelmReleases](docs/helmreleases.md)
