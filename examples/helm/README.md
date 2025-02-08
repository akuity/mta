The helm directory contains valid manifest files each of which
creates a `HelmRepository` and `HelmRelease` resource that you can
apply to your cluster and see how `mta` helps in converting those resources
into compatible Argo CD CRs.

## How to use mta

### Example 1:

In this example, we are going to use the `manifest-1.yaml` for reference. This manifest simply creates
a `HelmRepository` resource and `HelmRelease` resource in the `default` namespace. The `HelmRelease`
resource simply references a resource of kind `HelmRepository` and name `podinfo`.

Once you've applied the manifest:

1. Run the `mta scan` command to look for all the Kustomization and Helmreleases resources
    ```
    ┌───────────────┬─────────┬───────────┬────────────────────────────────────────────────────────────────────────┐
    │ KIND          │ NAME    │ NAMESPACE │ STATUS                                                                 │
    ├───────────────┼─────────┼───────────┼────────────────────────────────────────────────────────────────────────┤
    │ HelmRelease   │ podinfo │ default   │  Fulfilling prerequisites                                              │
    └───────────────┴─────────┴───────────┴────────────────────────────────────────────────────────────────────────┘
    ```

2. Convert the helmrelease resource into Argo CD compatible CR using the following command:
    ```
    mta helmrelease --name podinfo --namespace default
    ```
   This will generate valid Argo CD comptaible CRs for you to apply to the cluster.


### Example 2:

In this example, we are going to use the `manifest-2.yaml` for reference. This manifest simply creates
a `HelmRepository` resource in the `helm` namespace and `HelmRelease` resource in the `default` namespace.
The `HelmRelease` resource simply references a `HelmRepository` named `helmrepo` that we created above
in the `helm` namespace.

Once you've applied the manifest:

1. Run the `mta scan` command to look for all the Kustomization and HelmRelease resources
    ```
   ┌───────────────┬──────────────────┬───────────┬────────────────────────────────────────────────────────────────────────┐
   │ KIND          │ NAME             │ NAMESPACE │ STATUS                                                                 │
   ├───────────────┼──────────────────┼───────────┼────────────────────────────────────────────────────────────────────────┤
   │ HelmRelease   │ helmrelease      │ default   │ Applied revision: master@sha1:b99bf8c252d47db1cccfb6546aec650679645e61 │
   └───────────────┴──────────────────┴───────────┴────────────────────────────────────────────────────────────────────────┘
    ```

2. Convert the HelmRelease resource into Argo CD compatible CRs using the following command:
    ```
    mta helmrelease --name helmrelease --namespace default
    ```
   This will generate valid Argo CD comptaible CRs for you to apply to the cluster.

### Troubleshooting:
1. If you ever come across the following error message when converting to Argo CD compatible CRs

    ```
    FATA[0000] helmrepositories.source.toolkit.fluxcd.io "<helmrepository-name>" not found 
    ```

    Ensure that the `HelmRepository` that your `HelmRelease` resource references to exists in the namespace
    you've specified in the `sourceRef` field. If you've not specified a namespace in the `sourceRef` field, `mta` will search
    for the `HelmRepository` in `HelmRelease` resource namespace. For more information, refer to the:

    - [Helm Reference docs](https://fluxcd.io/flux/components/helm/helmreleases/).
    - [Helm API Reference docs](https://fluxcd.io/flux/components/helm/api/v2/)

2. If you don't specify the `--namespace` flag to `mta helmrelease` command , mta will try to look for a `HelmRelease` resource in the `flux-system` namespace.
Hence, If you get the following error message without specifying the namespace flag

    ```
    FATA[0000] helmreleases.helm.toolkit.fluxcd.io "<helmrelease-name>" not found 
    ```

    Ensure that the `HelmRelease` resource exists in the `flux-system` namespace.
