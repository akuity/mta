package templates

var ArgoCDMigrationYAMLOLD string = `apiVersion: v1
kind: Secret
metadata:
  name: mta-migration
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: repository
type: Opaque
data:
  sshPrivateKey: {{.SSHPrivateKey}}
  type: Z2l0
  url: {{.GitOpsRepoB64}}
---
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: mta-migration
  namespace: argocd
spec:
  destination:
    server: https://kubernetes.default.svc
  project: default
  source:
    path: {{.SourcePath}}
    repoURL: {{.GitOpsRepo}}
    targetRevision: {{.GitOpsRepoBranch}}
    directory:
      recurse: true
  syncPolicy:
    syncOptions:
    - Validate=false
    - CreateNamespace=true
    automated:
      prune: true
      selfHeal: true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
`
var ArgoCDMigrationYAML string = `apiVersion: v1
kind: Secret
metadata:
  name: mta-migration
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: repository
type: Opaque
data:
  sshPrivateKey: {{.SSHPrivateKey}}
  type: Z2l0
  url: {{.GitOpsRepoB64}}
---
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: mta-migration
  namespace: argocd
spec:
  generators:
  - git:
      repoURL: {{.GitOpsRepo}}
      revision: {{.GitOpsRepoBranch}}
      directories:
      - path: {{.SourcePath}}/*
      - path: {{.SourcePath}}/flux-system
        exclude: true
  template:
    metadata:
      name: {{.RawPathBasename}}
    spec:
      project: default
      syncPolicy:
        syncOptions:
        - Validate=false
        - CreateNamespace=true
        automated:
          prune: true
          selfHeal: true
        retry:
          limit: 5
          backoff:
            duration: 5s
            factor: 2
            maxDuration: 3m
      source:
        repoURL: {{.GitOpsRepo}}
        targetRevision: {{.GitOpsRepoBranch}}
        path: {{.RawPath}}
      destination:
        server: https://kubernetes.default.svc
`
