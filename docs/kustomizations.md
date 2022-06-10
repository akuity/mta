# Kustomization Quickstart

This is a quickstart so you can test how this migration tool works.

# Setup Flux

If you already have Flux running, you can skip ahead. If not, you'll need to install/bootstrap Flux on to your Kubernetes cluster.

You'll need

* [Flux CLI](https://github.com/fluxcd/flux2/releases)
* A GitHub token
* Kubernetes Cluster

Easiest way to test is with a [KIND cluster](https://github.com/kubernetes-sigs/kind)

```shell
kind create cluster
```

## Bootstrapping

The easiest way to get Flux up and running is to use the CLI to bootstrap. Best to use the [official documentation](https://fluxcd.io/docs/get-started/). But in Short:

Export your GitHub Token

```shell
export GITHUB_TOKEN=123abc456def789
```

Install/Configure Flux on the cluster

> *NOTE* replace the username with your GitHub username

```shell
flux bootstrap github --owner christianh814 --private --personal --repository flux-demo
```

## Add Applications

To test the migration, add some applications.

First, clone the repo from GitHub and `cd` into that repo's directory.

> *NOTE* Again, replace the username with your GitHub username

```shell
git clone git@github.com:christianh814/flux-demo.git
cd flux-demo
```

Add some applications (you can probably copy/paste this)

```shell
mkdir welcome-php
cd welcome-php
kubectl create ns welcome-php --dry-run=client -o yaml > welcome-php-ns.yaml
kubectl create deployment welcome-php -n welcome-php --image=quay.io/redhatworkshops/welcome-php:latest --dry-run=client -o yaml > welcome-php-deployment.yaml
kubectl create service clusterip welcome-php -n welcome-php --tcp=8080:8080 -o yaml --dry-run=client > welcome-php-service.yaml
kustomize create --autodetect --recursive --namespace welcome-php

cd ../

mkdir bgd
cd bgd
kubectl create deployment --image=quay.io/redhatworkshops/bgd:latest bgd -n bgd --dry-run=client -o yaml > bgd-deployment.yaml
kubectl create service clusterip bgd -n bgd --tcp=8080:8080 -o yaml --dry-run=client > bgd-service.yaml
kubectl create ns bgd --dry-run=client -o yaml > bgd-namespace.yaml
kustomize create --autodetect --recursive --namespace bgd

cd ../
```

You should have 3 directories. Two with the apps you've just created and one for the flux system.

```shell
$ tree -d .
.
├── bgd
├── flux-system
└── welcome-php
```

Commit and push

```shell
git add .
git commit -am "added test applications"
git push
```

Reconcile the changes 

```shell
flux reconcile source git flux-system
```

# Migrate

Migration happens via an ApplicationSet. To see what will be created, just run the migration tool.

> *NOTE* If youre Kustomizations are in a different namespace or named differnt, use `--namespace` and `--name` respectively.

```shell
mta kustomization
```

You can redirect this to a file or pipe it directly into `kubectl apply`.

First make sure Argo CD is installed

```shell
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

Then pipe it into `kubectl apply`

```shell
mta kustomization  | kubectl apply -n argocd -f -
```

This should create an ApplicationSet with two Applications (discarding the `flux-system` directory)

```shell
$ kubectl get appsets,apps -n argocd
NAME                                       AGE
applicationset.argoproj.io/mta-migration   53s

NAME                                  SYNC STATUS   HEALTH STATUS
application.argoproj.io/bgd           Synced        Healthy
application.argoproj.io/welcome-php   Synced        Healthy
```

Now suspend reconciliation on Flux

```shell
flux suspend kustomization --namespace flux-system flux-system
```

Once suspended, you can safely delete the Kustomization

```shell
flux delete kustomization flux-system  -s
```

It is now safe to delete Flux

```shell
flux uninstall  -s
```

The applications should still be running

```shell
kubectl get pods,svc,deploy -A  | egrep 'bgd|welcome-php'
```