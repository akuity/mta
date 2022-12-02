/*
Copyright © 2022 Christian Hernandez christian@chernand.io

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"os"
	"strings"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/christianh814/mta/pkg/argo"
	"github.com/christianh814/mta/pkg/utils"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/tools/clientcmd"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// kustomizationCmd represents the kustomization command
var kustomizationCmd = &cobra.Command{
	Use:     "kustomization",
	Aliases: []string{"k"},
	Short:   "Exports a Kustomization into an ApplicationSet",
	Long: `This is a migration tool that helps you move your Flux Kustomizations
into an Argo CD ApplicationSet. Example:

mta kustomization --name=mykustomization --namespace=flux-system | kubectl apply -n argocd -f -

This utilty exports the named Kustomization and the source Git repo and
creates a manifests to stdout, which you can pipe into an apply command
with kubectl.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get the Argo CD namespace
		argoCDNamespace, err := cmd.Flags().GetString("argocd-namespace")
		if err != nil {
			log.Fatal(err)
		}

		// Get the options from the CLI
		kubeConfig, err := cmd.Flags().GetString("kubeconfig")
		if err != nil {
			log.Fatal(err)
		}
		kustomizationName, _ := cmd.Flags().GetString("name")
		kustomizationNamespace, _ := cmd.Flags().GetString("namespace")
		confirmMigrate, _ := cmd.Flags().GetBool("confirm-migrate")

		// Set up the default context
		ctx := context.TODO()

		// Set up the scheme of components we need
		scheme := runtime.NewScheme()
		kustomizev1.AddToScheme(scheme)
		sourcev1.AddToScheme(scheme)
		corev1.AddToScheme(scheme)
		argov1alpha1.AddToScheme(scheme)

		// create rest config using the kubeconfig file.
		restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			log.Fatal(err)
		}

		// Create a new client based on the restconfig and scheme
		k, err := client.New(restConfig, client.Options{
			Scheme: scheme,
		})
		if err != nil {
			log.Fatal(err)
		}

		// get the kustomization based on the type, report if there's an error
		kustomization := &kustomizev1.Kustomization{}
		err = k.Get(ctx, types.NamespacedName{Namespace: kustomizationNamespace, Name: kustomizationName}, kustomization)
		if err != nil {
			log.Fatal(err)
		}

		// get the gitsource
		gitSource := &sourcev1.GitRepository{}
		err = k.Get(ctx, types.NamespacedName{Namespace: kustomizationNamespace, Name: kustomizationName}, gitSource)
		if err != nil {
			log.Fatal(err)
		}

		//Get the secret holding the info we need
		secret := &corev1.Secret{}
		err = k.Get(ctx, types.NamespacedName{Namespace: kustomizationNamespace, Name: gitSource.Spec.SecretRef.Name}, secret)
		if err != nil {
			log.Fatal(err)
		}

		//	Argo CD ApplicationSet is sensitive about how you give it paths in the Git Dir generator. We need to figure some things out
		var sourcePath string
		var sourcePathExclude string

		spl := strings.SplitAfter(kustomization.Spec.Path, "./")

		if len(spl[1]) == 0 {
			sourcePath = `*`
			sourcePathExclude = "flux-system"
		} else {
			sourcePath = spl[1] + "/*"
			sourcePathExclude = spl[1] + "/flux-system"
		}

		// Generate the ApplicationSet manifest based on the struct
		applicationSet := argo.GitDirApplicationSet{
			Namespace:               argoCDNamespace,
			GitRepoURL:              gitSource.Spec.URL,
			GitRepoRevision:         gitSource.Spec.Reference.Branch,
			GitIncludeDir:           sourcePath,
			GitExcludeDir:           sourcePathExclude,
			AppName:                 "{{path.basename}}",
			AppProject:              "default",
			AppRepoURL:              gitSource.Spec.URL,
			AppTargetRevision:       gitSource.Spec.Reference.Branch,
			AppPath:                 "{{path}}",
			AppDestinationServer:    "https://kubernetes.default.svc",
			AppDestinationNamespace: kustomization.Spec.TargetNamespace,
			SSHPrivateKey:           string(secret.Data["identity"]),
			GitOpsRepo:              gitSource.Spec.URL,
		}

		appset, err := argo.GenGitDirAppSet(applicationSet)
		if err != nil {
			log.Fatal(err)
		}

		// Generate the ApplicationSet Secret and set the GVK
		appsetSecret := utils.GenK8SSecret(applicationSet)

		// Do the migration automatically if that is set, if not print to stdout
		if confirmMigrate {
			// Suspend reconcilation
			kustomization.Spec.Suspend = true
			k.Update(ctx, kustomization)

			// Finally, create the ApplicationSet with the ApplicationSet Secret
			log.Info("Migrating Kustomization \"" + kustomization.Name + "\" to ArgoCD via an ApplicationSet")
			if err := utils.CreateK8SObjects(k, ctx, appsetSecret, appset); err != nil {
				log.Fatal(err)
			}

		} else {
			// Print the ApplicationSet and Secret to stdout

			// Set the printer type to YAML
			printr := printers.NewTypeSetter(k.Scheme()).ToPrinter(&printers.YAMLPrinter{})

			// Print the AppSet secret to Stdout
			if err := printr.PrintObj(appsetSecret, os.Stdout); err != nil {
				log.Fatal(err)
			}

			// print the AppSet YAML to Strdout
			if err := printr.PrintObj(appset, os.Stdout); err != nil {
				log.Fatal(err)
			}

		}

	},
}

func init() {
	rootCmd.AddCommand(kustomizationCmd)
	rootCmd.MarkPersistentFlagRequired("name")

	kustomizationCmd.Flags().Bool("confirm-migrate", false, "Automatically Migrate the Kustomization to an ApplicationSet")
}
