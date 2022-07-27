package format_test

import (
	"fmt"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/format"
)

func TestShortMessage(t *testing.T) {
	// given
	expectedAttachments := "```\n" +
		heredoc.Doc(`
			message 1
			message 2
			Recommendations:
			- recommendation 1
			- recommendation 2
			Warnings:
			- warning 1
			- warning 2
		`) + "```"
	testCases := []struct {
		Name     string
		Input    events.Event
		Expected string
	}{
		{
			Name: "Create event for cluster resource",
			Input: events.Event{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				Name:            "new-ns",
				Messages:        []string{"message 1", "message 2"},
				Type:            config.CreateEvent,
				Cluster:         "cluster-name",
				Recommendations: []string{"recommendation 1", "recommendation 2"},
				Warnings:        []string{"warning 1", "warning 2"},
			},
			Expected: fmt.Sprintf("Namespace *new-ns* has been created in *cluster-name* cluster\n%s", expectedAttachments),
		},
		{
			Name: "Update event for namespaced resource",
			Input: events.Event{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				Name:            "pod",
				Namespace:       "namespace",
				Messages:        []string{"message 1", "message 2"},
				Type:            config.CreateEvent,
				Cluster:         "cluster-name",
				Recommendations: []string{"recommendation 1", "recommendation 2"},
				Warnings:        []string{"warning 1", "warning 2"},
			},
			Expected: fmt.Sprintf("Pod *namespace/pod* has been created in *cluster-name* cluster\n%s", expectedAttachments),
		},
		{
			Name: "Error event for cluster resource",
			Input: events.Event{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				Name:            "new-ns",
				Messages:        []string{"message 1", "message 2"},
				Type:            config.ErrorEvent,
				Cluster:         "cluster-name",
				Recommendations: []string{"recommendation 1", "recommendation 2"},
				Warnings:        []string{"warning 1", "warning 2"},
			},
			Expected: fmt.Sprintf("Error occurred for Namespace *new-ns* in *cluster-name* cluster\n%s", expectedAttachments),
		},
		{
			Name: "Error event for namespaced resource",
			Input: events.Event{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				Name:            "pod",
				Namespace:       "namespace",
				Messages:        []string{"message 1", "message 2"},
				Type:            config.ErrorEvent,
				Cluster:         "cluster-name",
				Recommendations: []string{"recommendation 1", "recommendation 2"},
				Warnings:        []string{"warning 1", "warning 2"},
			},
			Expected: fmt.Sprintf("Error occurred for Pod *namespace/pod* in *cluster-name* cluster\n%s", expectedAttachments),
		},
		{
			Name: "Warning event for cluster resource",
			Input: events.Event{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				Name:            "new-ns",
				Messages:        []string{"message 1", "message 2"},
				Type:            config.WarningEvent,
				Cluster:         "cluster-name",
				Recommendations: []string{"recommendation 1", "recommendation 2"},
				Warnings:        []string{"warning 1", "warning 2"},
			},
			Expected: fmt.Sprintf("Warning for Namespace *new-ns* in *cluster-name* cluster\n%s", expectedAttachments),
		},
		{
			Name: "Warning event for namespaced resource",
			Input: events.Event{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				Name:            "pod",
				Namespace:       "namespace",
				Messages:        []string{"message 1", "message 2"},
				Type:            config.WarningEvent,
				Cluster:         "cluster-name",
				Recommendations: []string{"recommendation 1", "recommendation 2"},
				Warnings:        []string{"warning 1", "warning 2"},
			},
			Expected: fmt.Sprintf("Warning for Pod *namespace/pod* in *cluster-name* cluster\n%s", expectedAttachments),
		},
		{
			Name: "Info event for namespaced resource",
			Input: events.Event{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				Name:            "pod",
				Namespace:       "namespace",
				Messages:        []string{"message 1", "message 2"},
				Type:            config.InfoEvent,
				Cluster:         "cluster-name",
				Recommendations: []string{"recommendation 1", "recommendation 2"},
				Warnings:        []string{"warning 1", "warning 2"},
			},
			Expected: fmt.Sprintf("Info for Pod *namespace/pod* in *cluster-name* cluster\n%s", expectedAttachments),
		},
		{
			Name: "Info event for namespaced resource",
			Input: events.Event{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				Name:            "pod",
				Namespace:       "namespace",
				Messages:        []string{"message 1", "message 2"},
				Type:            config.InfoEvent,
				Cluster:         "cluster-name",
				Recommendations: []string{"recommendation 1", "recommendation 2"},
				Warnings:        []string{"warning 1", "warning 2"},
			},
			Expected: fmt.Sprintf("Info for Pod *namespace/pod* in *cluster-name* cluster\n%s", expectedAttachments),
		},
	}

	for _, tC := range testCases {
		t.Run(tC.Name, func(t *testing.T) {
			// when
			actual := format.ShortMessage(tC.Input)

			// then
			assert.Equal(t, tC.Expected, actual)
		})
	}
}
