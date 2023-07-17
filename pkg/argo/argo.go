package argo

import (
	"context"

	v1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ArgoCdGitApplicationSet is a struct that holds the ArgoCD Git ApplicationSet
// TODO: Make a Generic "ApplicationSet" struct that can be used generically (i.e. specify your generator)
type GitDirApplicationSet struct {
	Namespace               string
	GitRepoURL              string
	GitRepoRevision         string
	GitIncludeDir           string
	GitExcludeDir           []string
	AppName                 string
	AppProject              string
	AppRepoURL              string
	AppTargetRevision       string
	AppPath                 string
	AppDestinationServer    string
	AppDestinationNamespace string
	SSHPrivateKey           string
	GitOpsRepo              string
}

// ArgoCdApplication is a struct that holds the ArgoCD Application
type ArgoCdHelmApplication struct {
	Name                 string
	Namespace            string
	DestinationNamespace string
	DestinationServer    string
	Project              string
	HelmChart            string
	HelmRepo             string
	HelmTargetRevision   string
	HelmValues           string
	HelmCreateNamespace  string
}

// GenArgoCdApplication generates an ArgoCD Application
func GenArgoCdHelmApplication(app ArgoCdHelmApplication) (*v1alpha1.Application, error) {
	// Some Defaults
	// TODO: Make these configurable
	aSPAutomated := v1alpha1.SyncPolicyAutomated{Prune: true, SelfHeal: true}
	aSyncOptions := v1alpha1.SyncOptions{"CreateNamespace=" + app.HelmCreateNamespace, "Validate=false"}

	// Create Empty Application
	a := &v1alpha1.Application{}

	// Set GVK scheme
	a.SetGroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind("Application"))
	a.SetName(app.Name)
	a.SetNamespace(app.Namespace)
	a.Spec = v1alpha1.ApplicationSpec{
		Project: app.Project,
		Source: &v1alpha1.ApplicationSource{
			Chart:          app.HelmChart,
			RepoURL:        app.HelmRepo,
			TargetRevision: app.HelmTargetRevision,
			Helm: &v1alpha1.ApplicationSourceHelm{
				Values: app.HelmValues,
			},
		},
		Destination: v1alpha1.ApplicationDestination{
			Namespace: app.DestinationNamespace,
			Server:    app.DestinationServer,
		},
		SyncPolicy: &v1alpha1.SyncPolicy{
			Automated:   &aSPAutomated,
			SyncOptions: aSyncOptions,
		},
	}

	// Return the application def
	return a, nil
}

// GenGitDirApplicationSet generates an ArgoCD Git Directory ApplicationSet that
func GenGitDirAppSet(appSet GitDirApplicationSet) (*v1alpha1.ApplicationSet, error) {
	// Some Defaults
	// TODO: Make these configurable
	var TargetNamespace string
	asName := "mta-migration"
	asSyncOptions := v1alpha1.SyncOptions{"CreateNamespace=true", "Validate=false"}
	asSPAutomated := v1alpha1.SyncPolicyAutomated{Prune: true, SelfHeal: true}
	asRetry := v1alpha1.RetryStrategy{Limit: 5, Backoff: &v1alpha1.Backoff{Duration: "5s", Factor: func(i int64) *int64 { return &i }(2), MaxDuration: "3m"}}

	// Set the Target Namespace to "default" if it's not set
	if appSet.AppDestinationNamespace == "" {
		TargetNamespace = "default"
	} else {
		TargetNamespace = appSet.AppDestinationNamespace
	}

	// Create Empty ApplicationSet
	as := &v1alpha1.ApplicationSet{}

	// Set GVK scheme
	as.SetGroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind("ApplicationSet"))

	as.SetName(asName)
	as.SetNamespace(appSet.Namespace)
	//
	as.Spec.Generators = []v1alpha1.ApplicationSetGenerator{
		{
			Git: &v1alpha1.GitGenerator{
				RepoURL:  appSet.GitRepoURL,
				Revision: appSet.GitRepoRevision,
				Directories: []v1alpha1.GitDirectoryGeneratorItem{
					{Path: appSet.GitIncludeDir},
				},
				//Template: v1alpha1.ApplicationSetTemplate{},
			},
		},
	}

	// Append any excluded directories
	for _, d := range appSet.GitExcludeDir {
		as.Spec.Generators[0].Git.Directories = append(as.Spec.Generators[0].Git.Directories, v1alpha1.GitDirectoryGeneratorItem{Path: d, Exclude: true})
	}

	// Reset the Git Template spec because we aren't using it
	as.Spec.Generators[0].Git.Template.Reset()

	// Set up the Application template Spec
	as.Spec.Template = v1alpha1.ApplicationSetTemplate{
		ApplicationSetTemplateMeta: v1alpha1.ApplicationSetTemplateMeta{
			Name: appSet.AppName,
		},
		Spec: v1alpha1.ApplicationSpec{
			Project: appSet.AppProject,
			SyncPolicy: &v1alpha1.SyncPolicy{
				SyncOptions: asSyncOptions,
				Automated:   &asSPAutomated,
				Retry:       &asRetry,
			},
			Source: &v1alpha1.ApplicationSource{
				RepoURL:        appSet.AppRepoURL,
				TargetRevision: appSet.AppTargetRevision,
				Path:           appSet.AppPath,
			},
			Destination: v1alpha1.ApplicationDestination{
				Server:    appSet.AppDestinationServer,
				Namespace: TargetNamespace,
			},
		},
	}

	// Return ApplicationSet
	return as, nil
}

// IsArgoRunning checks if ArgoCD is running. Best effort as it just checks to see if the namespace exists
func IsArgoRunning(client client.Client, ns string) bool {
	// If we can't fine the namespace, return false
	namespaceobj := &apiv1.Namespace{}
	err := client.Get(context.Background(), types.NamespacedName{Name: ns}, namespaceobj)

	if err != nil && apierrors.IsNotFound(err) {
		return false
	}

	// If we're here we should be okay
	return true
}
