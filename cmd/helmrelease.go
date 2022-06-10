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
	"fmt"
	"os"

	yaml "sigs.k8s.io/yaml"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
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
		// Get the options from the CLI
		kubeConfig, err := cmd.Flags().GetString("kubeconfig")
		if err != nil {
			log.Fatal(err)
		}
		helmReleaseName, _ := cmd.Flags().GetString("name")
		helmReleaseNamespace, _ := cmd.Flags().GetString("namespace")

		// Set up the default context
		ctx := context.TODO()

		// Set up the schema because HelmRelease and Repo is a CRD
		scheme := runtime.NewScheme()
		helmv2.AddToScheme(scheme)
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

		/*
				WE MIGHT NOT NEED THIS
			// Create a standard client to get the secret from the Core API later
			sc, err := utils.NewClient(kubeConfig)
			if err != nil {
				log.Fatal(err)
			}
		*/

		//Get the helmrelease based on type, report if there's an error
		helmRelease := &helmv2.HelmRelease{}
		err = k.Get(ctx, client.ObjectKey{Namespace: helmReleaseNamespace, Name: helmReleaseName}, helmRelease)
		if err != nil {
			log.Fatal(err)
		}

		// The Values output is in JSON
		json, err := json.Marshal(helmRelease.Spec.Values)
		if err != nil {
			log.Fatal(err)
		}

		//Convert JSON Values to YAML
		yaml, err := yaml.JSONToYAML(json)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(yaml))

	},
}

func init() {
	rootCmd.AddCommand(helmreleaseCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// helmreleaseCmd.PersistentFlags().String("foo", "", "A help for foo")
	kcf, _ := os.UserHomeDir()
	helmreleaseCmd.Flags().String("kubeconfig", kcf+"/.kube/config", "Path to the kubeconfig file to use (if not the standard one).")
	helmreleaseCmd.Flags().String("name", "flux-system", "Name of HelmRelease to export")
	helmreleaseCmd.Flags().String("namespace", "flux-system", "Namespace of where the HelmRelease is")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// helmreleaseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
