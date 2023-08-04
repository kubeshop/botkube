package gh

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v53/github"
	"github.com/gregjones/httpcache"
	"golang.org/x/oauth2"

	"github.com/kubeshop/botkube/internal/httpx"
	multierrx "github.com/kubeshop/botkube/pkg/multierror"
)

type (
	ClientConfig struct {
		// Auth allows you to set either PAT or APP credentials.
		// If none provided then watch functionality could not work properly, e.g. you can reach the API calls quota or if you are setting GitHub Enterprise base URL then an unauthorized error can occur.
		Auth AuthConfig `yaml:"auth"`

		// The GitHub base URL for API requests. Defaults to the public GitHub API, but can be set to a domain endpoint to use with GitHub Enterprise.
		// Default: https://api.github.com/
		BaseURL string `yaml:"baseUrl"`

		// The GitHub upload URL for uploading files. It is taken into account only when the GITHUB_BASE_URL is also set. If only the GITHUB_BASE_URL is provided then this parameter defaults to the GITHUB_BASE_URL value.
		// Default: https://uploads.github.com/
		UploadURL string `yaml:"uploadUrl"`
	}

	// AuthConfig represents the authentication configuration.
	AuthConfig struct {
		// The GitHub access token.
		// Instruction for creating a token can be found here: https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/#creating-a-token.
		AccessToken string `yaml:"accessToken"`

		// AppConfig represents the GitHub App configuration.
		App AppConfig `yaml:"app"`
	}

	// AppConfig represents the GitHub App configuration.
	AppConfig struct {
		// GitHub App ID for authentication. This replaces the GITHUB_ACCESS_TOKEN.
		// Instruction for creating a GitHub App can be found here: https://github.com/mszostok/codeowners-validator/blob/main/docs/gh-token.md
		ID int64 `yaml:"id"`

		// GitHub App Installation ID. Required when GITHUB_APP_ID is set.
		InstallationID int64 `yaml:"installationId"`

		// GitHub App private key in PEM format. Required when GITHUB_APP_ID is set.
		PrivateKey string `yaml:"privateKey"`
	}
)

// Validate validates if provided client options are valid.
func (c *ClientConfig) Validate() error {
	if c.Auth.AccessToken != "" && c.Auth.App.ID != 0 {
		return errors.New("GitHub Access Token cannot be provided when App ID is specified")
	}

	issues := multierrx.New()
	if c.Auth.App.ID != 0 {
		if c.Auth.App.InstallationID == 0 {
			issues = multierrx.Append(issues, errors.New("GitHub App Installation ID is required with App ID"))
		}
		if c.Auth.App.PrivateKey == "" {
			issues = multierrx.Append(issues, errors.New("GitHub App Private Key is required with App ID"))
		}
	}

	return nil
}

func NewClient(cfg *ClientConfig) (*github.Client, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}

	httpClient := httpx.NewHTTPClient()
	httpClient.Transport = httpcache.NewMemoryCacheTransport()

	switch {
	case cfg.Auth.AccessToken != "":
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: cfg.Auth.AccessToken},
		)

		httpClient = &http.Client{
			Transport: &oauth2.Transport{
				Base:   httpcache.NewMemoryCacheTransport(),
				Source: oauth2.ReuseTokenSource(nil, ts),
			},
		}
	case cfg.Auth.App.ID != 0:
		httpClient, err = createAppInstallationHTTPClient(cfg)
		if err != nil {
			return nil, err
		}
	}

	httpClient.Timeout = httpx.DefaultTimeout

	baseURL, uploadURL := cfg.BaseURL, cfg.UploadURL

	httpClient.Transport = &LogRateLimitHeaders{RoundTripper: httpClient.Transport}
	if baseURL == "" {
		return github.NewClient(httpClient), nil
	}

	if uploadURL == "" { // often the baseURL is same as the uploadURL, so we do not require to provide both of them
		uploadURL = baseURL
	}

	bURL, uURL := httpx.CanonicalURLPath(baseURL), httpx.CanonicalURLPath(uploadURL)
	return github.NewEnterpriseClient(bURL, uURL, httpClient)
}

var headers = map[string]struct{}{
	"X-RateLimit-Limit":     {},
	"X-RateLimit-Remaining": {},
	"X-RateLimit-Reset":     {},
	"Retry-After":           {},
}

type LogRateLimitHeaders struct {
	http.RoundTripper
}

func (l LogRateLimitHeaders) RoundTrip(request *http.Request) (*http.Response, error) {
	resp, err := l.RoundTripper.RoundTrip(request)
	if resp != nil {
		values := map[string]string{}
		for name := range headers {
			values[name] = resp.Header.Get(name)
		}
		fmt.Printf("All headers: %v\n", values)
	}
	return resp, err
}

func createAppInstallationHTTPClient(cfg *ClientConfig) (client *http.Client, err error) {
	tr := httpcache.NewMemoryCacheTransport()
	itr, err := ghinstallation.New(tr, cfg.Auth.App.ID, cfg.Auth.App.InstallationID, []byte(cfg.Auth.App.PrivateKey))
	if err != nil {
		return nil, err
	}

	return &http.Client{Transport: itr}, nil
}
