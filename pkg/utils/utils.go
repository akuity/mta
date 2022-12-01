package utils

import (
	"context"
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/christianh814/mta/pkg/argo"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// GenK8SSecret generates a kubernetes secret using a clientset
func GenK8SSecret(a argo.GitDirApplicationSet) *apiv1.Secret {
	// Some Defaults
	// TODO: Make these configurable
	sName := "mta-migration"
	sLabels := map[string]string{
		"argocd.argoproj.io/secret-type": "repository",
	}

	sData := map[string]string{
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

// MigrateToArgoCD Creates Argo CD Applications
func MigrateToArgoCD(c client.Client, ctx context.Context, obj ...client.Object) error {
	// Migrate the objects
	for _, o := range obj {
		if err := c.Create(ctx, o); err != nil {
			return err
		}
	}

	// If we're here, it should have gone okay...
	return nil
}

// NewDynamicClient returns a dyamnic kubernetes interface
func NewDynamicClient(kubeConfigPath string) (dynamic.Interface, error) {
	// Try and find the kubeconfig path based on "normal" means
	if kubeConfigPath == "" {
		kubeConfigPath = os.Getenv("KUBECONFIG")
	}
	if kubeConfigPath == "" {
		kubeConfigPath = clientcmd.RecommendedHomeFile // use default path(.kube/config)
	}

	// build the dynamic client and return it
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(kubeConfig)
}

// NewClient returns a kubernetes interface
func NewClient(kubeConfigPath string) (kubernetes.Interface, error) {
	if kubeConfigPath == "" {
		kubeConfigPath = os.Getenv("KUBECONFIG")
	}
	if kubeConfigPath == "" {
		kubeConfigPath = clientcmd.RecommendedHomeFile // use default path(.kube/config)
	}
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(kubeConfig)
}

// WriteTemplate is a generic template writing mechanism
func WriteTemplate(tpl string, vars interface{}) error {
	tmpl := template.Must(template.New("").Funcs(sprig.GenericFuncMap()).Parse(tpl))
	err := tmpl.Execute(os.Stdout, vars)

	if err != nil {
		return err
	}
	// If we're here we should be okay
	return nil
}
