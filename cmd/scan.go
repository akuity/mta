/*
Copyright © 2022 Christian Hernandez <christian@chernand.io>

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

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/jedib0t/go-pretty/v6/table"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Looks for all HelmReleases and Kustomizations",
	Long: `Looks for HelmReleases and Kustomizations in the cluster and
displays the results.
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Set up table
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Kind", "Name", "Namespace", "Status"})

		// Set up the default context
		ctx := context.TODO()

		// Get the Kubeconfig to use
		kubeConfig, err := cmd.Flags().GetString("kubeconfig")
		if err != nil {
			log.Fatal(err)
		}

		// Set up the schema because HelmRelease and Kustomization are CRDs
		kScheme := runtime.NewScheme()
		kustomizev1.AddToScheme(kScheme)
		sourcev1.AddToScheme(kScheme)
		helmv2.AddToScheme(kScheme)
		sourcev1.AddToScheme(kScheme)

		// create rest config using the kubeconfig file.
		restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			log.Fatal(err)
		}

		// Create a new client based on the restconfig and scheme
		k, err := client.New(restConfig, client.Options{
			Scheme: kScheme,
		})
		if err != nil {
			log.Fatal(err)
		}

		// Get all Helm Releases in the cluster
		helmReleaseList := &helmv2.HelmReleaseList{}
		if err = k.List(ctx, helmReleaseList); err != nil {
			log.Fatal(err)
		}

		// Add all Helm Releases to the table
		for _, hr := range helmReleaseList.Items {
			t.AppendRow(table.Row{"HelmRelease", hr.Name, hr.Namespace, hr.Status.Conditions[0].Message})
		}

		// Add a separotor to the table
		t.AppendSeparator()

		// Get all Kustomizations in the cluster
		kustomizationList := &kustomizev1.KustomizationList{}
		if err = k.List(ctx, kustomizationList); err != nil {
			log.Fatal(err)
		}

		// Add all Kustomizations to the table
		for _, k := range kustomizationList.Items {
			t.AppendRow(table.Row{"Kustomization", k.Name, k.Namespace, k.Status.Conditions[0].Message})
		}

		//Render the table to the console
		t.SetStyle(table.StyleLight)
		t.Render()

	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
