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
	"encoding/base64"
	"strings"

	"github.com/christianh814/mta/pkg/utils"
	"github.com/christianh814/mta/vars/templates"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

		// Set up the default context
		ctx := context.TODO()

		// Set up the schema because Kustomization and GitRepo is a CRD
		scheme := runtime.NewScheme()
		kustomizev1.AddToScheme(scheme)
		sourcev1.AddToScheme(scheme)

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

		// Create a standard client to get the secret from the Core API later
		sc, err := utils.NewClient(kubeConfig)
		if err != nil {
			log.Fatal(err)
		}

		// get the kustomization based on the type, report if there's an error
		kustomization := &kustomizev1.Kustomization{}
		err = k.Get(ctx, client.ObjectKey{Namespace: kustomizationNamespace, Name: kustomizationName}, kustomization)
		if err != nil {
			log.Fatal(err)
		}

		// get the gitsource
		gitSource := &sourcev1.GitRepository{}
		err = k.Get(ctx, client.ObjectKey{Namespace: kustomizationNamespace, Name: kustomizationName}, gitSource)
		if err != nil {
			log.Fatal(err)
		}

		//Get the secret holding the info we need
		secret, err := sc.CoreV1().Secrets(kustomizationNamespace).Get(ctx, gitSource.Spec.SecretRef.Name, v1.GetOptions{})
		if err != nil {
			log.Fatal()
		}

		//	Argo CD ApplicationSet is sensitive about how you give it paths in the Git Dir generator. We need to figure some things out
		var sourcePath string
		var sourcePathExclude string

		spl := strings.SplitAfter(kustomization.Spec.Path, "./")

		if len(spl[1]) == 0 {
			sourcePath = `'*'`
			sourcePathExclude = "flux-system"
		} else {
			sourcePath = spl[1] + "/*"
			sourcePathExclude = spl[1] + "/flux-system"
		}

		// Generate Template YAML based on things we've figured out
		argoCDYAMLVars := struct {
			SSHPrivateKey     string
			GitOpsRepoB64     string
			SourcePath        string
			SourcePathExclude string
			GitOpsRepo        string
			GitOpsRepoBranch  string
			RawPathBasename   string
			RawPath           string
			ArgoCDNamespace   string
		}{
			SSHPrivateKey:     base64.StdEncoding.EncodeToString(secret.Data["identity"]),
			GitOpsRepoB64:     base64.StdEncoding.EncodeToString([]byte(gitSource.Spec.URL)),
			SourcePath:        sourcePath,
			SourcePathExclude: sourcePathExclude,
			GitOpsRepo:        gitSource.Spec.URL,
			GitOpsRepoBranch:  gitSource.Spec.Reference.Branch,
			RawPathBasename:   `'{{path.basename}}'`,
			RawPath:           `'{{path}}'`,
			ArgoCDNamespace:   argoCDNamespace,
		}
		//Send the YAML to stdout
		err = utils.WriteTemplate(templates.ArgoCDAppSetMigrationYAML, argoCDYAMLVars)
		if err != nil {
			log.Fatal(err)
		}

	},
}

func init() {
	rootCmd.AddCommand(kustomizationCmd)
	rootCmd.MarkPersistentFlagRequired("name")
}
