package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/kubeshop/botkube/cmd/cli/cmd/config"
	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
)

const (
	srvAddress         = "localhost:8085"
	loginURL           = "http://localhost:3000/cli/login?cli_server_login=http://localhost:8085/login_redirect"
	redirectURLSuccess = "http://localhost:3000/cli/login?success=true"
)

// NewLogin returns a cobra.Command for logging into a Botkube Cloud.
func NewLogin() *cobra.Command {
	login := &cobra.Command{
		Use:   "login [OPTIONS]",
		Short: "Login to a Botkube Cloud",
		Example: heredoc.WithCLIName(`
			# start interactive setup
			<cli> login
		`, cli.Name),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(cmd.Context(), os.Stdout)
		},
	}

	return login
}

func runLogin(_ context.Context, w io.Writer) error {
	t, err := fetchToken(srvAddress, loginURL)
	if err != nil {
		return err
	}

	c := config.Config{Token: t.Token}
	if err := c.Save(); err != nil {
		return err
	}

	okCheck := color.New(color.FgGreen).FprintlnFunc()
	okCheck(w, "Login Succeeded")

	return nil
}

type tokenResp struct {
	Token string `json:"token"`
}

func fetchToken(addr, authUrl string) (*tokenResp, error) {
	ch := make(chan *tokenResp)
	errCh := make(chan error)

	mux := http.NewServeMux()
	mux.HandleFunc("/login_redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectURLSuccess, http.StatusFound)

		ch <- &tokenResp{
			Token: r.URL.Query().Get("token"),
		}
	})

	s := http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	fmt.Println(heredoc.Docf(`
			Your browser has been opened to visit:
				 %s
		`, authUrl))
	err := browser.OpenURL(authUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to open page: %v", err)
	}

	select {
	case token := <-ch:
		_ = s.Shutdown(context.Background())
		return token, nil
	case err = <-errCh:
		return nil, err
	}
}
