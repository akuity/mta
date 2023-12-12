package utils

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/akuity/mta/pkg/argo"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	yaml "sigs.k8s.io/yaml"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/pkg/log"
	fluxuninstall "github.com/fluxcd/flux2/pkg/uninstall"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// MigrateKustomizationToApplicationSet migrates a Kustomization to an Argo CD ApplicationSet
func MigrateKustomizationToApplicationSet(c client.Client, ctx context.Context, ans string, k kustomizev1.Kustomization, exd []string) error {
	// excludedDirs will be paths excluded by the gidir generator
	excludedDirs := exd

	// Get the GitRepository from the Kustomization
	// get the gitsource
	gitSource := &sourcev1.GitRepository{}
	err := c.Get(ctx, types.NamespacedName{Namespace: k.Namespace, Name: k.Name}, gitSource)
	if err != nil {
		return err
	}

	//Get the secret holding the info we need
	//secret, err := _.CoreV1().Secrets(k.Namespace).Get(ctx, gitSource.Spec.SecretRef.Name, v1.GetOptions{})
	secret := &apiv1.Secret{}
	err = c.Get(ctx, types.NamespacedName{Namespace: k.Namespace, Name: gitSource.Spec.SecretRef.Name}, secret)
	if err != nil {
		return err
	}

	//Argo CD ApplicationSet is sensitive about how you give it paths in the Git Dir generator. We need to figure some things out
	var sourcePath string
	var sourcePathExclude string

	spl := strings.SplitAfter(k.Spec.Path, "./")

	if len(spl[1]) == 0 {
		sourcePath = `*`
		sourcePathExclude = "flux-system"
	} else {
		sourcePath = spl[1] + "/*"
		sourcePathExclude = spl[1] + "/flux-system"
	}

	// Add sourcePathExclude to the excludedDirs
	excludedDirs = append(excludedDirs, sourcePathExclude)

	// Generate the ApplicationSet manifest based on the struct
	applicationSet := argo.GitDirApplicationSet{
		Namespace:               ans,
		GitRepoURL:              gitSource.Spec.URL,
		GitRepoRevision:         gitSource.Spec.Reference.Branch,
		GitIncludeDir:           sourcePath,
		GitExcludeDir:           excludedDirs,
		AppName:                 "{{path.basename}}",
		AppProject:              "default",
		AppRepoURL:              gitSource.Spec.URL,
		AppTargetRevision:       gitSource.Spec.Reference.Branch,
		AppPath:                 "{{path}}",
		AppDestinationServer:    "https://kubernetes.default.svc",
		AppDestinationNamespace: k.Spec.TargetNamespace,
		SSHPrivateKey:           string(secret.Data["identity"]),
		GitOpsRepo:              gitSource.Spec.URL,
	}

	appset, err := argo.GenGitDirAppSet(applicationSet)
	if err != nil {
		return err
	}

	// Generate the ApplicationSet Secret and set the GVK
	appsetSecret := GenK8SSecret(applicationSet)

	// Suspend Kustomization reconcilation
	if err := SuspendFluxObject(c, ctx, &k); err != nil {
		return err

	}

	// Suspend git repo reconcilation
	if err := SuspendFluxObject(c, ctx, gitSource); err != nil {
		return err
	}

	// Finally, create the Argo CD Application
	if err := CreateK8SObjects(c, ctx, appsetSecret, appset); err != nil {
		return err
	}

	// Delete the Kustomization
	if err := DeleteK8SObjects(c, ctx, &k); err != nil {
		return err
	}

	// Delete the GitRepository
	if err := DeleteK8SObjects(c, ctx, gitSource); err != nil {
		return err
	}

	// If we're here, it should have gone okay...
	return nil
}

// MigrateHelmReleaseToApplication migrates a HelmRelease to an Argo CD Application
func MigrateHelmReleaseToApplication(c client.Client, ctx context.Context, ans string, h helmv2.HelmRelease) error {
	// Get the helmchart based on type, report if error
	helmRepo := &sourcev1.HelmRepository{}
	helmChart := &sourcev1.HelmChart{}
	err := c.Get(ctx, types.NamespacedName{Namespace: h.Namespace, Name: h.Spec.Chart.Spec.SourceRef.Name}, helmRepo)
	if err != nil {
		return err
	}
	err = c.Get(ctx, types.NamespacedName{Namespace: h.Namespace, Name: h.Namespace + "-" + h.Name}, helmChart)
	if err != nil {
		return err
	}

	// Get the Values from the HelmRelease
	yaml, err := yaml.Marshal(h.Spec.Values)
	if err != nil {
		return err
	}

	// Generate the Argo CD Helm Application
	helmApp := argo.ArgoCdHelmApplication{
		Name:                 h.Spec.TargetNamespace + "-" + h.Name,
		Namespace:            ans,
		DestinationNamespace: h.Spec.TargetNamespace,
		DestinationServer:    "https://kubernetes.default.svc",
		Project:              "default",
		HelmChart:            h.Spec.Chart.Spec.Chart,
		HelmRepo:             helmRepo.Spec.URL,
		HelmTargetRevision:   h.Spec.Chart.Spec.Version,
		HelmValues:           string(yaml),
		HelmCreateNamespace:  strconv.FormatBool(h.Spec.Install.CreateNamespace),
	}

	helmArgoCdApp, err := argo.GenArgoCdHelmApplication(helmApp)
	if err != nil {
		return err
	}

	// Suspend helm reconcilation
	if err := SuspendFluxObject(c, ctx, &h); err != nil {
		return err
	}

	// Suspend helm repo reconcilation
	if err := SuspendFluxObject(c, ctx, helmRepo); err != nil {
		return err
	}

	// Suspend helm repo reconcilation
	if err := SuspendFluxObject(c, ctx, helmChart); err != nil {
		return err
	}

	// Finally, create the Argo CD Application
	if err := CreateK8SObjects(c, ctx, helmArgoCdApp); err != nil {
		return err
	}

	// Delete the HelmRelease
	if err := DeleteK8SObjects(c, ctx, &h); err != nil {
		return err
	}

	// Delete the HelmRepository
	if err := DeleteK8SObjects(c, ctx, helmRepo); err != nil {
		return err
	}

	// Delete the HelmChart
	if err := DeleteK8SObjects(c, ctx, helmChart); err != nil {
		return err
	}

	// If we're here, it should have gone okay...
	return nil
}

// FluxCleanUp cleans up flux resources
func FluxCleanUp(k client.Client, ctx context.Context, log log.Logger, ns string) error {
	// Set up the context with timeout
	cwt, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	//Set up the flux uninstall options
	// TODO: Maybe make these configurable
	uninstallFlags := struct {
		keepNamespace bool
		dryRun        bool
		silent        bool
	}{
		keepNamespace: false,
		dryRun:        false,
		silent:        false,
	}

	// Uninstall the components
	if err := fluxuninstall.Components(cwt, log, k, ns, uninstallFlags.dryRun); err != nil {
		return err
	}

	// Uninstall the finalizers
	if err := fluxuninstall.Finalizers(cwt, log, k, uninstallFlags.dryRun); err != nil {
		return err
	}

	// Uninstall CRDS
	if err := fluxuninstall.CustomResourceDefinitions(cwt, log, k, uninstallFlags.dryRun); err != nil {
		return err
	}

	// Uninstall the namespace
	if err := fluxuninstall.Namespace(cwt, log, k, ns, uninstallFlags.dryRun); err != nil {
		return err
	}

	// If we're here, it should have gone okay...
	return nil
}

// SuspendFluxObject suspends Flux specific objects based on the schema passed in the client.
func SuspendFluxObject(c client.Client, ctx context.Context, obj ...client.Object) error {
	// suspend the objects
	for _, o := range obj {
		if err := c.Patch(ctx, o, client.RawPatch(types.MergePatchType, []byte(`{"spec":{"suspend":true}}`))); err != nil {
			return err
		}
	}

	// If we're here, it should have gone okay...
	return nil
}

// CreateK8SObjects Creates Kubernetes Objects on the Cluster based on the schema passed in the client.
func CreateK8SObjects(c client.Client, ctx context.Context, obj ...client.Object) error {
	// Migrate the objects
	for _, o := range obj {
		if err := c.Create(ctx, o); err != nil {
			return err
		}
	}

	// If we're here, it should have gone okay...
	return nil
}

// DeleteK8SObjects Deletes Kubernetes Objects on the Cluster based on the schema passed in the client.
func DeleteK8SObjects(c client.Client, ctx context.Context, obj ...client.Object) error {
	// Migrate the objects
	for _, o := range obj {
		if err := c.Delete(ctx, o); err != nil {
			return err
		}
	}

	// If we're here, it should have gone okay...
	return nil
}

// GenK8SSecret generates a kubernetes secret using a clientset
func GenK8SSecret(a argo.GitDirApplicationSet) *apiv1.Secret {
	// Some Defaults
	// TODO: Make these configurable
	sData := map[string]string{}
	sName := "mta-migration"
	sLabels := map[string]string{
		"argocd.argoproj.io/secret-type": "repository",
	}

	sData = map[string]string{
		"sshPrivateKey": a.SSHPrivateKey,
		"type":          "git",
		"url":           a.GitOpsRepo,
	}

	// Create the secret
	s := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sName,
			Namespace: a.Namespace,
			Labels:    sLabels,
		},
		Type:       apiv1.SecretTypeOpaque,
		StringData: sData,
	}

	// set the gvk for the secret
	s.SetGroupVersionKind(apiv1.SchemeGroupVersion.WithKind("Secret"))

	// Return the secret
	return s

}

// NewRestClient returns a rest.Config
func NewRestConfig(kubeConfigPath string) (*rest.Config, error) {
	if kubeConfigPath == "" {
		kubeConfigPath = os.Getenv("KUBECONFIG")
	}
	if kubeConfigPath == "" {
		kubeConfigPath = clientcmd.RecommendedHomeFile // use default path(.kube/config)
	}
	return clientcmd.BuildConfigFromFlags("", kubeConfigPath)
}
