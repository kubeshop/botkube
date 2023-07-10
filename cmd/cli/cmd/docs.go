package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/kubeshop/botkube/internal/cli/frontmatter"
)

const (
	docsTargetDir = "./cmd/cli/docs"
)

// NewDocs returns a cobra.Command for generating Botkube CLI documentation.
func NewDocs() *cobra.Command {
	return &cobra.Command{
		Use:    "gen-usage-docs",
		Hidden: true,
		Short:  "Generate usage documentation",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := NewRoot()
			root.DisableAutoGenTag = true

			defaultLinkHandler := func(s string) string { return s }
			return doc.GenMarkdownTreeCustom(root, docsTargetDir, frontmatter.FilePrepender, defaultLinkHandler)
		},
	}
}
