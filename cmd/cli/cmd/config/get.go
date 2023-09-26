package config

import (
	"fmt"
	"os"
	"reflect"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/analytics"
	"github.com/kubeshop/botkube/internal/cli/config"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
	"github.com/kubeshop/botkube/internal/cli/printer"
	"github.com/kubeshop/botkube/internal/kubex"
)

type GetOptions struct {
	OmitEmpty bool
	Exporter  config.ExporterOptions
}

// NewGet returns a cobra.Command for getting Botkube configuration.
func NewGet() *cobra.Command {
	var opts GetOptions

	resourcePrinter := printer.NewForResource(os.Stdout, printer.WithJSON(), printer.WithYAML())

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Displays Botkube configuration",
		Example: heredoc.WithCLIName(`
			# Show configuration for currently installed Botkube
			<cli> config get
			
			# Show configuration in JSON format
			<cli> config get -ojson

			# Save configuration in file
			<cli> config get > config.yaml
		`, cli.Name),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			status := printer.NewStatus(cmd.ErrOrStderr(), "Fetching Botkube configuration")
			defer func() {
				status.End(err == nil)
			}()

			k8sCfg, err := kubex.LoadRestConfigWithMetaInformation()
			if err != nil {
				return fmt.Errorf("while creating k8s config: %w", err)
			}

			err = status.InfoStructFields("Export details:", exportDetails{
				ExporterVersion: opts.Exporter.Tag,
				K8sCtx:          k8sCfg.CurrentContext,
				LookupNamespace: opts.Exporter.BotkubePodNamespace,
				LookupPodLabel:  opts.Exporter.BotkubePodLabel,
			})
			if err != nil {
				return err
			}

			cfg, botkubeVersionStr, err := config.GetFromCluster(cmd.Context(), status, k8sCfg.K8s, opts.Exporter, false)
			if err != nil {
				return fmt.Errorf("while getting configuration: %w", err)
			}

			var raw interface{}
			err = yaml.Unmarshal(cfg, &raw)
			if err != nil {
				return fmt.Errorf("while loading configuration: %w", err)
			}

			if opts.OmitEmpty {
				status.Step("Removing empty keys from configuration")
				raw = removeEmptyValues(raw)
				status.End(true)
			}

			status.Infof("Exported Botkube configuration (agent version: %q)", botkubeVersionStr)

			return resourcePrinter.Print(raw)
		},
	}

	cmd = analytics.InjectAnalyticsReporting(*cmd, "config get")

	flags := cmd.Flags()

	flags.BoolVar(&opts.OmitEmpty, "omit-empty-values", true, "Omits empty keys from printed configuration")

	opts.Exporter.RegisterFlags(flags)

	resourcePrinter.RegisterFlags(flags)

	return cmd
}

type exportDetails struct {
	K8sCtx          string `pretty:"Kubernetes Context"`
	ExporterVersion string `pretty:"Exporter Version"`
	LookupNamespace string `pretty:"Lookup Namespace"`
	LookupPodLabel  string `pretty:"Lookup Pod Label"`
}

func removeEmptyValues(obj any) any {
	switch v := obj.(type) {
	case map[string]any:
		newObj := make(map[string]any)
		for key, value := range v {
			if value != nil {
				newValue := removeEmptyValues(value)
				if newValue != nil {
					newObj[key] = newValue
				}
			}
		}
		if len(newObj) == 0 {
			return nil
		}
		return newObj
	default:
		val := reflect.ValueOf(v)
		if val.IsZero() {
			return nil
		}
		return obj
	}
}
