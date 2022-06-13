# HelmRelease Quickstart

This is a quickstart so you can test how this migration tool works.

# Setup Flux

If you already have Flux running, you can [skip ahead](#migrate). If not, you'll need to install/bootstrap Flux on to your Kubernetes cluster.

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

## Add Helm Application

To test the migration, add some applications.

First, clone the repo from GitHub and `cd` into that repo's directory.

> *NOTE* Again, replace the username with your GitHub username

```shell
git clone git@github.com:christianh814/flux-demo.git
cd flux-demo/flux-system
```

Create HelmSource

```shell
flux create source helm redhat-helm-charts \
--url=https://redhat-developer.github.io/redhat-helm-charts --interval=1m \
--export > source-helm-repo-rh.yaml
```

Next, create a values file for this HelmRelease

```shell
cat <<EOF > values.yaml
build:
  enabled: false
deploy:
  route:
    enabled: false
image:
  name: quay.io/ablock/gitops-helm-quarkus
EOF
```

Next, create a Helmrelease base on this `values.yaml` file

```shell
flux create helmrelease sample \
--values ./values.yaml \
--interval=1m \
--source=HelmRepository/redhat-helm-charts \
--chart=quarkus --chart-version=0.0.3 --target-namespace quarkus --create-target-namespace \
--export > source-helmrepo-sample-quarkus.yaml
```

No need to commit the `values.yaml` file

```shell
rm values.yaml
```

Commit and push

```shell
git add .
git commit -am "added test application"
git push
```

Reconcile the changes 

```shell
flux reconcile source git flux-system
```

# Migrate

Migration happens via an Argo CD Application. To see what will be created, just run the migration tool.

> *NOTE* If your HelmReleases are in a different namespace or named differnt, use `--namespace` and `--name` respectively. Note that `--name` is required.

```shell
mta helmrelease --name=myhelmrelease
```

You can redirect this to a file or pipe it directly into `kubectl apply`.

First make sure Argo CD is installed

```shell
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

Then pipe it into `kubectl apply`

```shell
mta helmrelease --name=myhelmrelease | kubectl apply -n argocd -f -
```

This should create an Application based on your HelmRelease

```shell
$ kubectl get apps -n argocd
NAME             SYNC STATUS   HEALTH STATUS
quarkus-sample   Synced        Healthy
```

Now suspend reconciliation on Flux

```shell
flux suspend kustomization --namespace flux-system flux-system
```

Once suspended, you can safely delete the Kustomization that Holds the HelmRelease.

```shell
flux delete kustomization flux-system  -s
```

It is now safe to delete Flux

```shell
flux uninstall  -s
```

The applications should still be running

```shell
kubectl get pods,svc,deploy -A  | egrep 'quarkus'
```
