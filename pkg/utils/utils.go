package utils

import (
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

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
