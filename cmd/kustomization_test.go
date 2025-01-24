package cmd

import (
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/magiconair/properties/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestKustomizeGitRepoNamespace(t *testing.T) {
	tests := []struct {
		name                     string
		kustomization            *kustomizev1.Kustomization
		expectedGitRepoNamespace string
	}{
		{
			name: "when git repository namespace is defined",
			kustomization: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "kustomization-namespace",
				},
				Spec: kustomizev1.KustomizationSpec{
					SourceRef: kustomizev1.CrossNamespaceSourceReference{
						Namespace: "gitrepo-namespace",
					},
				},
			},
			expectedGitRepoNamespace: "gitrepo-namespace",
		},
		{
			name: "when git repository namespace is not defined",
			kustomization: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "kustomization-namespace",
				},
				Spec: kustomizev1.KustomizationSpec{
					SourceRef: kustomizev1.CrossNamespaceSourceReference{},
				},
			},
			expectedGitRepoNamespace: "kustomization-namespace",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitRepoNamespace := getGitRepoNamespace(tt.kustomization)
			assert.Equal(t, gitRepoNamespace, tt.expectedGitRepoNamespace)
		})
	}
}
