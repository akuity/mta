package cmd

import (
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/magiconair/properties/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestHelmGetRepoNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	helmv2.AddToScheme(scheme)

	helmReleaseWithNamespace := &helmv2.HelmRelease{
		Spec: helmv2.HelmReleaseSpec{
			Chart: helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Namespace: "custom-namespace",
					},
				},
			},
		},
	}

	namespace := GetHelmRepoNamespace(helmReleaseWithNamespace)
	assert.Equal(t, "custom-namespace", namespace, "Expected the namespace to be 'custom-namespace'")

	helmReleaseWithoutNamespace := &helmv2.HelmRelease{
		Spec: helmv2.HelmReleaseSpec{
			Chart: helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					SourceRef: helmv2.CrossNamespaceObjectReference{},
				},
			},
		},
	}

	namespace = GetHelmRepoNamespace(helmReleaseWithoutNamespace)
	assert.Equal(t, helmReleaseWithoutNamespace.Namespace, namespace, "Expected the namespace to be 'default'")
}
