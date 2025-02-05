package cmd

import (
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestGetHelmRepoNamespace(t *testing.T) {
	tests := []struct {
		name              string
		helmRelease       *helmv2.HelmRelease
		helmRepoNamespace string
	}{
		{
			name: "when helmrelease namespace is defined",
			helmRelease: &helmv2.HelmRelease{
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Namespace: "custom-namespace",
							},
						},
					},
				},
			},
			helmRepoNamespace: "custom-namespace",
		},
		{
			name: "when helmrelease namespace is not defined",
			helmRelease: &helmv2.HelmRelease{
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							SourceRef: helmv2.CrossNamespaceObjectReference{},
						},
					},
				},
			},
			helmRepoNamespace: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace := GetHelmRepoNamespace(tt.helmRelease)
			assert.Equal(t, namespace, tt.helmRepoNamespace)
		})
	}
}
