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
	"encoding/json"
	"os"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	yaml "sigs.k8s.io/yaml"

	"github.com/christianh814/mta/pkg/argo"
	"github.com/christianh814/mta/pkg/utils"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/tools/clientcmd"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// helmreleaseCmd represents the helmrelease command
var helmreleaseCmd = &cobra.Command{
	Use:     "helmrelease",
	Aliases: []string{"HelmRelease", "hr"},
	Short:   "Exports a HelmRelease into an Application",
	Long: `This migration tool helps you move your Flux HelmReleases into Argo CD
Applications. Example:

mta helmrelease --name=myhelmrelease --namespace=flux-system | kubectl apply -n argocd -f -

This utilty exports the named HelmRelease and the source Helm repo and
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
		helmReleaseName, _ := cmd.Flags().GetString("name")
		helmReleaseNamespace, _ := cmd.Flags().GetString("namespace")
		confirmMigrate, _ := cmd.Flags().GetBool("confirm-migrate")

		// Set up the default context
		ctx := context.TODO()

		// Set up the schema because HelmRelease and Repo is a CRD
		scheme := runtime.NewScheme()
		helmv2.AddToScheme(scheme)
		sourcev1.AddToScheme(scheme)
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

		//Get the helmrelease based on type, report if there's an error
		helmRelease := &helmv2.HelmRelease{}
		err = k.Get(ctx, client.ObjectKey{Namespace: helmReleaseNamespace, Name: helmReleaseName}, helmRelease)
		if err != nil {
			log.Fatal(err)
		}

		// Get the helmchart based on type, report if error
		helmRepo := &sourcev1.HelmRepository{}
		err = k.Get(ctx, client.ObjectKey{Namespace: helmReleaseNamespace, Name: helmRelease.Spec.Chart.Spec.SourceRef.Name}, helmRepo)
		if err != nil {
			log.Fatal(err)
		}

		// The Values to the Helm chart output is in JSON
		json, err := json.Marshal(helmRelease.Spec.Values)
		if err != nil {
			log.Fatal(err)
		}

		//Convert JSON Values to YAML
		yaml, err := yaml.JSONToYAML(json)
		if err != nil {
			log.Fatal(err)
		}

		// Createnamespace comes out as a Bool, need to convert into a string
		var helmCreateNamespace string
		if helmRelease.Spec.Install.CreateNamespace {
			helmCreateNamespace = "true"
		} else {
			helmCreateNamespace = "false"
		}

		// Generate the Argo CD Helm Application
		helmApp := argo.ArgoCdHelmApplication{
			Name:                 helmRelease.Spec.Chart.Spec.Chart + "-" + helmRelease.Name,
			Namespace:            argoCDNamespace,
			DestinationNamespace: helmRelease.Spec.TargetNamespace,
			DestinationServer:    "https://kubernetes.default.svc",
			Project:              "default",
			HelmChart:            helmRelease.Spec.Chart.Spec.Chart,
			HelmRepo:             helmRepo.Spec.URL,
			HelmTargetRevision:   helmRelease.Spec.Chart.Spec.Version,
			HelmValues:           string(yaml),
			HelmCreateNamespace:  helmCreateNamespace,
		}

		helmArgoCdApp, err := argo.GenArgoCdHelmApplication(helmApp)
		if err != nil {
			log.Fatal(err)
		}

		// Do the migration automatically if that is set, if not print to stdout
		if confirmMigrate {
			log.Info("Migrating HelmRelease \"" + helmRelease.Name + "\" to Argo CD via an Application")
			// TODO: Suspend reconcilation
			if err := utils.MigrateToArgoCD(k, ctx, helmArgoCdApp); err != nil {
				log.Fatal(err)
			}
		} else {
			// Set the printer type to YAML
			printr := printers.NewTypeSetter(k.Scheme()).ToPrinter(&printers.YAMLPrinter{})

			// print the AppSet YAML to Strdout
			if err := printr.PrintObj(helmArgoCdApp, os.Stdout); err != nil {
				log.Fatal(err)
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(helmreleaseCmd)
	rootCmd.MarkPersistentFlagRequired("name")

	helmreleaseCmd.Flags().Bool("confirm-migrate", false, "Automatically Migrate the HelmRelease to an ApplicationSet")
}
