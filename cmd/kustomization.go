/*
Copyright © 2022 Christian Hernandez christian@email.com

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
	"fmt"
	"os"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// kustomizationCmd represents the kustomization command
var kustomizationCmd = &cobra.Command{
	Use:   "kustomization",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Set up the default context
		ctx := context.TODO()

		// Set up the schema because Kustomization is a CRD
		scheme := runtime.NewScheme()
		kustomizev1.AddToScheme(scheme)

		// Get the kubeconfig file if supplied
		kubeConfig, err := cmd.Flags().GetString("kubeconfig")
		if err != nil {
			log.Fatal(err)
		}

		// create client
		/*
			kubeClient, err := utils.NewClient(kubeConfig)
			if err != nil {
				log.Fatal(err)
			}
		*/

		// create rest config
		restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			log.Fatal(err)
		}

		// Create a new client based on the restconfig and scheme
		c, err := client.New(restConfig, client.Options{
			Scheme: scheme,
		})
		if err != nil {
			log.Fatal(err)
		}

		// get the kustomization based on the type
		kustomization := &kustomizev1.Kustomization{}
		c.Get(ctx, client.ObjectKey{Namespace: "flux-system", Name: "flux-system"}, kustomization)
		fmt.Printf("Name: " + kustomization.Name + "\nPath: " + kustomization.Spec.Path + "\n")

	},
}

func init() {
	rootCmd.AddCommand(kustomizationCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// kustomizationCmd.PersistentFlags().String("foo", "", "A help for foo")
	kcf, _ := os.UserHomeDir()
	kustomizationCmd.PersistentFlags().String("kubeconfig", kcf+"/.kube/config", "Path to the kubeconfig file to use (if not the standard one).")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// kustomizationCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
