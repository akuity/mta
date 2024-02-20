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

	"github.com/akuity/mta/pkg/argo"
	"github.com/akuity/mta/pkg/utils"
	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	fluxlog "github.com/fluxcd/flux2/pkg/log"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/manifoldco/promptui"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		// Get excluded-dirs from the cli
		exd, err := cmd.Flags().GetStringSlice("exclude-dirs")
		if err != nil {
			log.Fatal(err)
		}

		// Set up the default context
		ctx := context.TODO()

		// Get the Kubeconfig to use
		kubeConfig, err := cmd.Flags().GetString("kubeconfig")
		if err != nil {
			log.Fatal(err)
		}

		// Get automigrate option from the cli
		autoMigrate, err := cmd.Flags().GetBool("auto-migrate")
		if err != nil {
			log.Fatal(err)
		}
		// Get automigrate option from the cli
		confirmMigrate, err := cmd.Flags().GetBool("confirm")
		if err != nil {
			log.Fatal(err)
		}

		// Get the Argo CD namespace in case of auto-migrate
		argoCDNamespace, err := cmd.Flags().GetString("argocd-namespace")
		if err != nil {
			log.Fatal(err)
		}

		// Set up the schema because HelmRelease and Kustomization are CRDs
		kScheme := runtime.NewScheme()
		kustomizev1.AddToScheme(kScheme)
		sourcev1.AddToScheme(kScheme)
		helmv2.AddToScheme(kScheme)
		sourcev1.AddToScheme(kScheme)
		corev1.AddToScheme(kScheme)
		argov1alpha1.AddToScheme(kScheme)

		// create rest config using the kubeconfig file.
		restConfig, err := utils.NewRestConfig(kubeConfig)
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

		// Get all Kustomizations in the cluster
		kustomizationList := &kustomizev1.KustomizationList{}
		if err = k.List(ctx, kustomizationList); err != nil {
			log.Fatal(err)
		}

		// Automigrate if the flag is set, otherwise just display the table
		if autoMigrate {
			// Prompt user to confirm migration
			if !confirmMigrate {
				prompt := promptui.Prompt{
					Label:     "Are you sure you want to migrate to Argo CD and uninstall Flux?",
					IsConfirm: true,
				}

				_, err := prompt.Run()

				if err != nil {
					log.Info("Automigration Cancelled")
					os.Exit(0)
				}
			} else {
				// Confirmation of migration has been confirmed
				log.Info("Auto-migration confirmed")
			}

			// Check if Argo CD is installed/running
			if !argo.IsArgoRunning(k, argoCDNamespace) {
				log.Fatal("Argo CD is not installed or running")
			}

			// Migrate Kustomizations
			for _, kl := range kustomizationList.Items {
				log.Info("Migrating Kustomization ", kl.Name)
				if err := utils.MigrateKustomizationToApplicationSet(k, ctx, argoCDNamespace, kl, exd); err != nil {
					log.Fatal(err)
				}
			}

			// Migrate HelmReleases
			for _, hl := range helmReleaseList.Items {
				log.Info("Migrating HelmRelease ", hl.Name)
				if err := utils.MigrateHelmReleaseToApplication(k, ctx, argoCDNamespace, hl); err != nil {
					log.Fatal(err)
				}
			}

			// Once we're done, we can uninstall Flux
			log.Info("Uninstalling Flux")
			if err := utils.FluxCleanUp(k, ctx, fluxlog.NopLogger{}, "flux-system"); err != nil {
				log.Fatal(err)
			}
		} else {

			// Set up table
			t := table.NewWriter()
			t.SetOutputMirror(os.Stdout)
			t.AppendHeader(table.Row{"Kind", "Name", "Namespace", "Status"})

			// Add all Helm Releases to the table
			for _, hr := range helmReleaseList.Items {
				t.AppendRow(table.Row{hr.Kind, hr.Name, hr.Namespace, utils.TruncMsg(hr.Status.Conditions[0].Message)})
			}

			// Add a separotor to the table
			t.AppendSeparator()

			// Add all Kustomizations to the table
			for _, k := range kustomizationList.Items {
				t.AppendRow(table.Row{k.Kind, k.Name, k.Namespace, utils.TruncMsg(k.Status.Conditions[0].Message)})
			}

			//Render the table to the console
			t.SetStyle(table.StyleLight)
			t.Render()

		}

	},
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().Bool("auto-migrate", false, "Migrate HelmReleases and Kustomizations to Argo CD and uninstalls Flux")
	scanCmd.Flags().Bool("confirm", false, "Confirm migraton to Argo CD and uninstalls Flux")
	scanCmd.Flags().StringSlice("exclude-dirs", []string{}, "Additional Directories (besides flux-system) to exclude from the GitDir generator. Can be single or comma separated")
}
