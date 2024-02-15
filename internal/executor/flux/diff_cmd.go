package flux

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/allegro/bigcache/v3"
	"github.com/bombsimon/logrusr/v4"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/gookit/color"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/kubeshop/botkube/pkg/plugin"
)

const (
	defaultNamespace                       = "flux-system"
	defaultGHDiffCommentHeader             = "Merging this pull request will introduce the following changes:"
	ghHDiffCommentHeaderWithClusterNameTpl = "Merging this pull request will trigger updates to the %s cluster, introducing the following changes:"

	requiredGHRefErrMsg = "The --github-ref flag is required to perform diff operation. It can be one of: pr number, full pr url, or branch name."
)

var gitHubDiffCommentTpl = heredoc.Doc(`
		#### Flux Kustomization changes
		
		%s
		
		<details open><summary>Output</summary>
		<p>
		
		
		%s
		
		</p>
		</details> 
		
		
		_Comment created via [Botkube Flux](https://docs.botkube.io/configuration/executor/flux) integration._`)

// DiffCommand holds diff sub-commands. We use it to customize the execution process.
type DiffCommand struct {
	KustomizationCommandAliases
	GitHub *struct {
		Comment *struct {
			URL        string `arg:"--url"`
			ArtifactID string `arg:"--cache-id"`
		} `arg:"subcommand:comment"`
	} `arg:"subcommand:gh"`
	Artifact *struct {
		Tool []string `arg:"positional"`
	} `arg:"subcommand:artifact"`
}

// KustomizeDiffCmdService provides functionality to run the flux diff kustomization process with Botkube related enhancements such as GitHub integration.
type KustomizeDiffCmdService struct {
	log   logrus.FieldLogger
	cache *bigcache.BigCache
}

// NewKustomizeDiffCmdService returns a new KustomizeDiffCmdService instance.
func NewKustomizeDiffCmdService(cache *bigcache.BigCache, log logrus.FieldLogger) *KustomizeDiffCmdService {
	return &KustomizeDiffCmdService{
		log:   log,
		cache: cache,
	}
}

// ShouldHandle returns true if commands should be handled by this service.
func (k *KustomizeDiffCmdService) ShouldHandle(command string) (*DiffCommand, bool) {
	if !strings.Contains(command, "diff") {
		return nil, false
	}

	var diffCmd struct {
		Diff *DiffCommand `arg:"subcommand:diff"`
	}

	err := plugin.ParseCommand(PluginName, command, &diffCmd)
	if err != nil {
		// if we cannot parse, it means that unknown command was specified
		k.log.WithError(err).Debug("Cannot parse input command into diff ones.")
		return nil, false
	}

	if diffCmd.Diff == nil {
		return nil, false
	}
	return diffCmd.Diff, true
}

// Run consumes the output of ShouldHandle method and runs specified command.
func (k *KustomizeDiffCmdService) Run(ctx context.Context, diffCmd *DiffCommand, kubeConfigPath string, kubeConfigBytes []byte, cfg Config) (executor.ExecuteOutput, error) {
	switch {
	case diffCmd.Artifact != nil:
		return executor.ExecuteOutput{}, errors.New("artifact diffing is not supported")
	case diffCmd.KustomizationCommandAliases.Get() != nil:
		return k.runKustomizeDiff(ctx, diffCmd, kubeConfigPath, kubeConfigBytes, cfg)
	case diffCmd.GitHub != nil && diffCmd.GitHub.Comment != nil:
		return k.postGitHubComment(ctx, diffCmd, cfg)
	default:
		return executor.ExecuteOutput{}, errors.New("unknown command")
	}
}

func (k *KustomizeDiffCmdService) postGitHubComment(ctx context.Context, diffCmd *DiffCommand, cfg Config) (executor.ExecuteOutput, error) {
	data, err := k.cache.Get(diffCmd.GitHub.Comment.ArtifactID)
	switch {
	case err == nil:
	case errors.Is(err, bigcache.ErrEntryNotFound):
		return executor.ExecuteOutput{
			Message: api.Message{
				Sections: []api.Section{
					{
						Base: api.Base{
							Header:      "❗ Missing report",
							Description: "The Kustomize diff report is missing from the cache. Please re-run the `flux diff ks` command to get a fresh report.",
						},
					},
				},
			},
		}, nil
	default:
		return executor.ExecuteOutput{}, fmt.Errorf("while getting diff data from cache: %w", err)
	}
	gh := NewGitHubCmdService(k.log)

	postPRCommentCmd := fmt.Sprintf("flux gh pr comment '%s' --body-file -", diffCmd.GitHub.Comment.URL)
	cmd, can := gh.ShouldHandle(postPRCommentCmd)
	if !can {
		return executor.ExecuteOutput{}, fmt.Errorf("command %q was not recognized by gh command executor", postPRCommentCmd)
	}

	header := defaultGHDiffCommentHeader
	clusterName := k.tryToResolveClusterName()
	if clusterName != "" {
		header = fmt.Sprintf(ghHDiffCommentHeaderWithClusterNameTpl, clusterName)
	}

	commentBody := fmt.Sprintf(gitHubDiffCommentTpl, header, formatx.CodeBlock(string(data)))
	return gh.Run(ctx, cmd, cfg, plugin.ExecuteCommandStdin(strings.NewReader(commentBody)))
}

func (k *KustomizeDiffCmdService) runKustomizeDiff(ctx context.Context, diffCmd *DiffCommand, kubeConfigPath string, kubeConfigBytes []byte, cfg Config) (executor.ExecuteOutput, error) {
	kustomizeDiff := diffCmd.KustomizationCommandAliases.Get()

	if kustomizeDiff.GitHubRef == "" {
		return executor.ExecuteOutput{}, errors.New(requiredGHRefErrMsg)
	}

	workdir, err := k.cloneResources(ctx, kustomizeDiff, kubeConfigBytes, cfg)
	if err != nil {
		return executor.ExecuteOutput{}, err
	}
	defer os.RemoveAll(workdir)

	out, changesDetected, err := k.runDiffCmd(ctx, kustomizeDiff.ToCmdString(),
		plugin.ExecuteCommandEnvs(map[string]string{
			"KUBECONFIG": kubeConfigPath,
		}),
		plugin.ExecuteCommandWorkingDir(workdir),
	)
	if err != nil {
		k.log.WithError(err).WithField("command", kustomizeDiff.ToCmdString()).Error("Failed to run command.")
		return executor.ExecuteOutput{}, fmt.Errorf("while running command: %v", err)
	}

	textFields, buttons := k.tryToGetPRDetails(ctx, out, kustomizeDiff, workdir, cfg)

	if !changesDetected && out == "" {
		return executor.ExecuteOutput{
			Message: api.Message{
				Sections: []api.Section{
					{
						Base: api.Base{
							Header: "No changes detected",
							Body: api.Body{
								Plaintext: "Running flux diff has not detected any changes that will be made by this pull request.",
							},
						},
						TextFields: textFields,
					},
					{
						Buttons: buttons,
					},
				},
			},
		}, nil
	}

	return executor.ExecuteOutput{
		Message: api.Message{
			Sections: []api.Section{
				{
					Base: api.Base{
						Header:      "⚠️ Changes detected",
						Description: "GitHub pull request highlights",
						Body:        api.Body{CodeBlock: out},
					},
					TextFields: textFields,
					Buttons:    buttons,
				},
			},
		},
	}, nil
}

func (k *KustomizeDiffCmdService) tryToGetPRDetails(ctx context.Context, out string, diff *KustomizationDiffCommand, workdir string, cfg Config) ([]api.TextField, []api.Button) {
	resolvePRDetailsCmd := fmt.Sprintf("gh pr view %s --json author,state,url", diff.GitHubRef)
	rawDetails, err := ExecuteCommand(ctx, resolvePRDetailsCmd,
		plugin.ExecuteCommandEnvs(map[string]string{
			"GH_TOKEN": cfg.GitHub.Auth.AccessToken,
		}),
		plugin.ExecuteCommandWorkingDir(workdir),
	)
	if err != nil {
		k.log.WithError(err).Debug("while getting pull request details")
		return nil, nil
	}

	var prDetails PRDetails
	err = json.Unmarshal([]byte(rawDetails), &prDetails)
	if err != nil {
		k.log.WithError(err).Debug("while unmarshalling pull request details")
		return nil, nil
	}

	textFields := api.TextFields{
		{Key: "Author", Value: formatx.AdaptiveCodeBlock(prDetails.Author.Login)},
		{Key: "State", Value: formatx.AdaptiveCodeBlock(prDetails.State)},
	}

	btnBuilder := api.NewMessageButtonBuilder()

	var btns api.Buttons

	if cfg.GitHub.Auth.AccessToken != "" { // if we don't have access token then we won't be able to create a comment or approve PR
		btns = k.appendPostDiffBtn(out, btns, btnBuilder, prDetails)
		btns = append(btns, btnBuilder.ForCommandWithoutDesc("Approve pull request", fmt.Sprintf("flux gh pr review %s --approve", prDetails.URL)))
	}

	btns = append(btns, btnBuilder.ForURL("View pull request", prDetails.URL))

	return textFields, btns
}

func (k *KustomizeDiffCmdService) appendPostDiffBtn(out string, btns api.Buttons, btnBuilder *api.ButtonBuilder, prDetails PRDetails) api.Buttons {
	if out == "" { // no diff
		return btns
	}
	cacheID, err := k.storeDiff(out)
	if err != nil {
		k.log.WithError(err).Info("Cannot store diff, skipping post as comment button")
		return btns
	}

	cmd := fmt.Sprintf("flux diff gh comment --url %s --cache-id %s --bk-cmd-header='Post diff as GitHub comment'", prDetails.URL, cacheID)

	return append(btns, btnBuilder.ForCommandWithoutDesc("Post diff under pull request", cmd, api.ButtonStylePrimary))
}

func (k *KustomizeDiffCmdService) storeDiff(out string) (string, error) {
	h := sha256.New()
	h.Write([]byte(out))
	cacheID := fmt.Sprintf("%x", h.Sum(nil))
	return cacheID, k.cache.Set(cacheID, []byte(out))
}

func (*KustomizeDiffCmdService) runDiffCmd(ctx context.Context, in string, opts ...plugin.ExecuteCommandMutation) (string, bool, error) {
	out, err := plugin.ExecuteCommand(ctx, in, opts...)
	if err != nil {
		if out.ExitCode == 1 { // the diff commands returns 1 if changes are detected
			return out.Stdout, true, nil
		}
		return "", false, err
	}

	message := strings.TrimSpace(color.ClearCode(out.CombinedOutput()))
	if message == "" {
		return "", false, nil
	}

	return message, false, nil
}

func resolveGitHubRepoURL(ctx context.Context, logger logrus.FieldLogger, kubeConfigBytes []byte, ns string, name string) (string, error) {
	scheme := runtime.NewScheme()
	if err := kustomizev1.AddToScheme(scheme); err != nil {
		return "", fmt.Errorf("while adding Kustomize scheme: %w", err)
	}

	if err := sourcev1.AddToScheme(scheme); err != nil {
		return "", fmt.Errorf("while adding Source scheme: %w", err)
	}

	kubeConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfigBytes)
	if err != nil {
		return "", fmt.Errorf("while reading kube config. %v", err)
	}

	log.SetLogger(logrusr.New(logger))

	cl, err := client.New(kubeConfig, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return "", fmt.Errorf("while creating client: %w", err)
	}

	// Resolve Kustomize
	ks := kustomizev1.Kustomization{}
	err = cl.Get(ctx, client.ObjectKey{
		Namespace: ns,
		Name:      name,
	}, &ks)

	if err != nil {
		return "", fmt.Errorf("while getting Kustomization: %w", err)
	}

	if ks.Spec.SourceRef.Kind != "GitRepository" {
		return "", nil // skip
	}

	// Get Kustomization GitHub repository
	git := sourcev1.GitRepository{}
	namespace := ks.Spec.SourceRef.Namespace
	if namespace == "" {
		namespace = ks.Namespace
	}

	err = cl.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      ks.Spec.SourceRef.Name,
	}, &git)

	if err != nil {
		return "", fmt.Errorf("while getting GitRepository: %w", err)
	}

	return git.Spec.URL, nil
}

func (k *KustomizeDiffCmdService) cloneResources(ctx context.Context, diff *KustomizationDiffCommand, kubeConfigBytes []byte, cfg Config) (string, error) {
	if diff.Namespace == "" {
		diff.Namespace = defaultNamespace
	}

	url, err := resolveGitHubRepoURL(ctx, k.log, kubeConfigBytes, diff.Namespace, diff.AppName)
	if err != nil {
		return "", err
	}

	// it may occur that it won't be a GitHub repository, but we proceed anyway.
	opts := []plugin.ExecuteCommandMutation{
		plugin.ExecuteCommandEnvs(map[string]string{
			"GH_TOKEN": cfg.GitHub.Auth.AccessToken,
		}),
	}

	dir, err := os.MkdirTemp(cfg.TmpDir.GetDirectory(), "gh-repo-")
	if err != nil {
		return "", fmt.Errorf("while writing creating tmp dir for repository: %w", err)
	}

	cloneCmd := fmt.Sprintf("gh repo clone %s %s -- --depth 1", url, dir)
	k.log.WithField("githubCmd", cloneCmd).Debug("Cloning GitHub repository...")
	_, err = plugin.ExecuteCommand(ctx, cloneCmd, opts...)
	if err != nil {
		return "", err
	}

	opts = append(opts, plugin.ExecuteCommandWorkingDir(dir))

	gitSetupOpts := append(opts, plugin.ExecuteCommandDependencyDir("")) // we want to execute a globally installed binary.
	// because we clone with --depth 1 we have issues as described here: https://github.com/cli/cli/issues/4287
	_, err = plugin.ExecuteCommand(ctx, `git config remote.origin.fetch "+refs/heads/*:refs/remotes/origin/*"`, gitSetupOpts...)
	if err != nil {
		return "", fmt.Errorf("while setting up git repo: %w", err)
	}

	checkoutCmd := fmt.Sprintf("gh pr checkout %s", diff.GitHubRef)
	k.log.WithField("checkoutCmd", checkoutCmd).Debug("Checking out pull request")
	_, err = plugin.ExecuteCommand(ctx, checkoutCmd, opts...)
	if err != nil {
		return "", err
	}

	return dir, nil
}

func (k *KustomizeDiffCmdService) tryToResolveClusterName() string {
	var apiCfg clientcmdapi.Config

	err := yaml.Unmarshal(nil, &apiCfg)
	if err != nil {
		k.log.WithError(err).Debug("Cannot unmarshal kubeconfig. Skipping obtaining cluster name")
		return ""
	}

	if apiCfg.CurrentContext == "default" {
		return "" // default was specified in the previous botkube version, for our use-case it's not useful so we skip it too.
	}
	return apiCfg.CurrentContext
}
