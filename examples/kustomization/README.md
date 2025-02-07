The kustomization directory contains valid manifest files each of which
creates a `GitRepository` and `Kustomization` resource that you can
apply to your cluster and see how `mta` helps in converting those resources
into compatible Argo CD CRs.

## How to use mta

### Example 1:

In this example, we are going to use the `manifest-1.yaml` for reference. This manifest simply creates
a `GitRepository` resource and `Kustomization` resource in the `default` namespace. The `Kustomization`
resource simply references a resource of kind `GitRepository` and name `podinfo`.

Once you've applied the manifest:

1. Run the `mta scan` command to look for all the Kustomization and HelmRelease resources
    ```
    ┌───────────────┬─────────┬───────────┬────────────────────────────────────────────────────────────────────────┐
    │ KIND          │ NAME    │ NAMESPACE │ STATUS                                                                 │
    ├───────────────┼─────────┼───────────┼────────────────────────────────────────────────────────────────────────┤
    │ Kustomization │ podinfo │ default   │ Applied revision: master@sha1:b99bf8c252d47db1cccfb6546aec650679645e61 │
    └───────────────┴─────────┴───────────┴────────────────────────────────────────────────────────────────────────┘
    ```

2. Convert the Kustomization resource into Argo CD compatible CR using the following command:
    ```
    mta kustomization --name podinfo --namespace default
    ```
   This will generate valid Argo CD comptaible CRs for you to apply to the cluster.


### Example 2:

In this example, we are going to use the `manifest-2.yaml` for reference. This manifest simply creates
a `GitRepository` resource in the `git` namespace and `Kustomization` resource in the `default` namespace.
The `Kustomization` resource simply references a `GitRepository` named `git-repository` that we created above
in the `git` namespace.

Once you've applied the manifest:

1. Run the `mta scan` command to look for all the Kustomization and Helmreleases resources
    ```
   ┌───────────────┬──────────────────┬───────────┬────────────────────────────────────────────────────────────────────────┐
   │ KIND          │ NAME             │ NAMESPACE │ STATUS                                                                 │
   ├───────────────┼──────────────────┼───────────┼────────────────────────────────────────────────────────────────────────┤
   │ Kustomization │ my-kustomization │ default   │ Applied revision: master@sha1:b99bf8c252d47db1cccfb6546aec650679645e61 │
   └───────────────┴──────────────────┴───────────┴────────────────────────────────────────────────────────────────────────┘
    ```

2. Convert the Kustomization resource into Argo CD compatible CRs using the following command:
    ```
    mta kustomization --name my-kustomization --namespace default
    ```
   This will generate valid Argo CD comptaible CRs for you to apply to the cluster.

### Troubleshooting:
1. If you ever come across the following error message when converting to Argo CD compatible CRs

    ```
    FATA[0000] gitrepositories.source.toolkit.fluxcd.io "<gitrepository-name>" not found 
    ```

   Ensure that the `GitRepository` that your `Kustomization` resources references to exists in the namespace
   you've specified in the `sourceRef` field. If you've not specified a namespace in the `sourceRef` namespace, `mta` will search
   for the `GitRepository` in `Kustomization` resource namespace. For more information, refer to the:

    - [Kustomization Reference docs](https://fluxcd.io/flux/components/kustomize/kustomizations/).
    - [Kustomization API Reference docs](https://fluxcd.io/flux/components/kustomize/api/v1/)

2. If you don't specify the `--namespace` flag to `mta kustomization` command, mta will try to look for a `Kustomization` resource in the `flux-system` namespace.
   Hence, If you get the following error message without specifying the namespace flag

    ```
    FATA[0000] kustomizations.kustomize.toolkit.fluxcd.io "<kustomization-name>" not found 
    ```

   Ensure that the `Kustomization` resource exists in the `flux-system` namespace.
