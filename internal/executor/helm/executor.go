package helm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/mattn/go-shellwords"
	"helm.sh/helm/v3/pkg/cli/values"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
)

type helm struct {
	Install   *InstallCmd `arg:"subcommand:install"`
	Namespace string      `arg:"--namespace,-n"`
}

var _ executor.Executor = &Executor{}

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (Executor) Metadata(context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     "version",
		Description: "TBD",
	}, nil
}

// install, uninstall, upgrade, rollback, list, version, test, status
//   ensure multiline commands work properly

// Execute returns a given command as response.
func (Executor) Execute(_ context.Context, in executor.ExecuteInput) (executor.ExecuteOutput, error) {
	cfg, err := MergeConfigs(in.Configs)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}

	_ = cfg

	k8sCfg, err := config.GetConfig()
	if err != nil {
		return executor.ExecuteOutput{}, err
	}

	var args helm
	p, err := arg.NewParser(arg.Config{
		Program: "helm",
	}, &args)
	if err != nil {
		return executor.ExecuteOutput{}, err
	}

	argsss, err := shellwords.Parse(in.Command)
	if err != nil {
		return executor.ExecuteOutput{}, err
	}
	fmt.Println(argsss)
	err = p.Parse(argsss)
	switch err {
	case nil, arg.ErrHelp, arg.ErrVersion:
		// ignore
		fmt.Println("err", err)
	default:
		fmt.Println(p.SubcommandNames())
		return executor.ExecuteOutput{}, err
	}

	actionConfig, err := NewActionConfig(k8sCfg, args.Namespace)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while creating action configuration: %w", err)
	}

	switch {
	case args.Install != nil:
		if err == arg.ErrHelp {
			p.WriteHelp(os.Stdout)
			return executor.ExecuteOutput{
				Data: "help",
			}, nil
		}

		err := args.Install.Validate()
		if err != nil {
			return executor.ExecuteOutput{}, err
		}

		client := newInstallClient(args.Install, actionConfig)
		valueOpts := &values.Options{
			ValueFiles:   args.Install.Values,
			StringValues: args.Install.SetString,
			Values:       args.Install.Set,
			FileValues:   args.Install.SetFile,
			JSONValues:   args.Install.SetJSON,
		}
		client.Namespace = args.Namespace

		var installArgs []string
		if args.Install.Name != "" {
			installArgs = append(installArgs, args.Install.Name)
		}
		if args.Install.Chart != "" {
			installArgs = append(installArgs, args.Install.Chart)
		}

		var buff strings.Builder
		rel, err := runInstall(context.Background(), installArgs, client, valueOpts, &buff)
		if err != nil {
			return executor.ExecuteOutput{}, err
		}

		fmt.Println(rel.Info.Notes)
	}
	return executor.ExecuteOutput{
		Data: "data",
	}, nil
}

// not supported install flags:
//      --atomic                                     if set, the installation process deletes the installation on failure. The --wait flag will be set automatically if --atomic is used
//      --ca-file string                             verify certificates of HTTPS-enabled servers using this CA bundle
//      --cert-file string                           identify HTTPS client using this SSL certificate file
//  -h, --help                                       help for install
//      --key-file string                            identify HTTPS client using this SSL key file
//      --keyring string                             location of public keys used for verification (default "/Users/mszostok/.gnupg/pubring.gpg")
//      --set-file stringArray                       set values from respective files specified via the command line (can specify multiple or separate values with commas: key1=path1,key2=path2)
//  -f, --values strings                             specify values in a YAML file or a URL (can specify multiple)
//      --wait                                       if set, will wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment, StatefulSet, or ReplicaSet are in a ready state before marking the release as successful. It will wait for as long as --timeout
//      --wait-for-jobs                              if set and --wait enabled, will wait until all Jobs have been completed before marking the release as successful. It will wait for as long as --timeout

// not supported Global flags:
//      --kube-apiserver string           the address and the port for the Kubernetes API server
//      --kube-as-group stringArray       group to impersonate for the operation, this flag can be repeated to specify multiple groups.
//      --kube-as-user string             username to impersonate for the operation
//      --kube-ca-file string             the certificate authority file for the Kubernetes API server connection
//      --kube-context string             name of the kubeconfig context to use
//      --kube-insecure-skip-tls-verify   if true, the Kubernetes API server's certificate will not be checked for validity. This will make your HTTPS connections insecure
//      --kube-tls-server-name string     server name to use for Kubernetes API server certificate validation. If it is not provided, the hostname used to contact the server is used
//      --kube-token string               bearer token used for authentication
//      --kubeconfig string               path to the kubeconfig file
//      --registry-config string          path to the registry config file (default "/Users/mszostok/Library/Preferences/helm/registry/config.json")
//      --repository-cache string         path to the file containing cached repository indexes (default "/Users/mszostok/Library/Caches/helm/repository")
//      --repository-config string        path to the file containing repository names and URLs (default "/Users/mszostok/Library/Preferences/helm/repositories.yaml")

// Flags:
//      --create-namespace                           create the release namespace if not present
//      --dependency-update                          update dependencies if they are missing before installing the chart
//      --description string                         add a custom description
//      --devel                                      use development versions, too. Equivalent to version '>0.0.0-0'. If --version is set, this is ignored
//      --disable-openapi-validation                 if set, the installation process will not validate rendered templates against the Kubernetes OpenAPI Schema
//      --dry-run                                    simulate an install
//  -g, --generate-name                              generate the name (and omit the NAME parameter)
//      --insecure-skip-tls-verify                   skip tls certificate checks for the chart download
//      --name-template string                       specify template used to name the release
//      --no-hooks                                   prevent hooks from running during install
//      --pass-credentials                           pass credentials to all domains
//      --password string                            chart repository password where to locate the requested chart
//      --post-renderer postRendererString           the path to an executable to be used for post rendering. If it exists in $PATH, the binary will be used, otherwise it will try to look for the executable at the given path
//      --post-renderer-args postRendererArgsSlice   an argument to the post-renderer (can specify multiple) (default [])
//      --render-subchart-notes                      if set, render subchart notes along with the parent
//      --replace                                    re-use the given name, only if that name is a deleted release which remains in the history. This is unsafe in production
//      --repo string                                chart repository url where to locate the requested chart
//      --set stringArray                            set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)
//      --set-json stringArray                       set JSON values on the command line (can specify multiple or separate values with commas: key1=jsonval1,key2=jsonval2)
//      --set-string stringArray                     set STRING values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)
//      --skip-crds                                  if set, no CRDs will be installed. By default, CRDs are installed if not already present
//      --timeout duration                           time to wait for any individual Kubernetes operation (like Jobs for hooks) (default 5m0s)
//      --username string                            chart repository username where to locate the requested chart
//      --verify                                     verify the package before using it
//      --version string                             specify a version constraint for the chart version to use. This constraint can be a specific tag (e.g. 1.1.1) or it may reference a valid range (e.g. ^2.0.0). If this is not specified, the latest version is used
//  -o, --output format                              prints the output in the specified format. Allowed values: table, json, yaml (default table)
//
//Global Flags:
//      --burst-limit int                 client-side default throttling limit (default 100)
//      --debug                           enable verbose output
//  -n, --namespace string                namespace scope for this request
