package login

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/browser"

	"github.com/kubeshop/botkube/cmd/cli/cmd/config"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
)

const (
	loginURLFmt           = "%s/cli/login?redirect_url=http://%s/login_redirect"
	redirectURLSuccessFmt = "%s/cli/login?success=true"
)

func Run(ctx context.Context, w io.Writer, opts Options) error {
	loginURL := fmt.Sprintf(loginURLFmt, opts.CloudDashboardURL, opts.LocalServerAddress)
	successURL := fmt.Sprintf(redirectURLSuccessFmt, opts.CloudDashboardURL)

	t, err := runServer(ctx, opts.LocalServerAddress, loginURL, successURL)
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

func runServer(ctx context.Context, srvAddr, authURL, successURL string) (*tokenResp, error) {
	ch := make(chan *tokenResp)
	errCh := make(chan error, 2)

	mux := http.NewServeMux()
	mux.HandleFunc("/login_redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, successURL, http.StatusFound)

		ch <- &tokenResp{
			Token: r.URL.Query().Get("token"),
		}
	})

	s := http.Server{
		Addr:              srvAddr,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	go func(ctx context.Context) {
		<-ctx.Done()
		fmt.Println("Shutting down server...")
		_ = s.Shutdown(context.Background())
		errCh <- errors.New("login process has been cancelled")
	}(ctx)

	fmt.Println(heredoc.Docf(`
			If your browser didn't open automatically, visit the URL to finish the login process:
				 %s
		`, authURL))
	err := browser.OpenURL(authURL)
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
