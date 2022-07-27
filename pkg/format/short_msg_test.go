package format_test

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/kubeshop/botkube/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/format"
)

func TestShortMessage(t *testing.T) {
	// given
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
			Expected: "Namespace *new-ns* has been created in *cluster-name* cluster\n" +
				"```\n" + heredoc.Doc(`
				message 1
				message 2
				Recommendations:
				- recommendation 1
				- recommendation 2
				Warnings:
				- warning 1
				- warning 2
			`) + "```",
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
			Expected: "Pod *namespace/pod* has been created in *cluster-name* cluster\n" +
				"```\n" + heredoc.Doc(`
				message 1
				message 2
				Recommendations:
				- recommendation 1
				- recommendation 2
				Warnings:
				- warning 1
				- warning 2
			`) + "```",
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
			Expected: "Error Occurred in Namespace: *new-ns* in *cluster-name* cluster\n" +
				"```\n" + heredoc.Doc(`
				message 1
				message 2
				Recommendations:
				- recommendation 1
				- recommendation 2
				Warnings:
				- warning 1
				- warning 2
			`) + "```",
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
			Expected: "Error Occurred in Pod: *namespace/pod* in *cluster-name* cluster\n" +
				"```\n" + heredoc.Doc(`
				message 1
				message 2
				Recommendations:
				- recommendation 1
				- recommendation 2
				Warnings:
				- warning 1
				- warning 2
			`) + "```",
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
			Expected: "Warning Namespace: *new-ns* in *cluster-name* cluster\n" +
				"```\n" + heredoc.Doc(`
				message 1
				message 2
				Recommendations:
				- recommendation 1
				- recommendation 2
				Warnings:
				- warning 1
				- warning 2
			`) + "```",
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
			Expected: "Pod Info: *namespace/pod* in *cluster-name* cluster\n" +
				"```\n" + heredoc.Doc(`
				message 1
				message 2
				Recommendations:
				- recommendation 1
				- recommendation 2
				Warnings:
				- warning 1
				- warning 2
			`) + "```",
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
			Expected: "Pod Info: *namespace/pod* in *cluster-name* cluster\n" +
				"```\n" + heredoc.Doc(`
				message 1
				message 2
				Recommendations:
				- recommendation 1
				- recommendation 2
				Warnings:
				- warning 1
				- warning 2
			`) + "```",
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
