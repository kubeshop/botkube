package grpcx

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kubeshop/botkube/pkg/config"
)

// ClientTransportCredentials returns gRPC client transport credentials based on the provided configuration.
func ClientTransportCredentials(log logrus.FieldLogger, cfg config.GRPCServer) (credentials.TransportCredentials, error) {
	if cfg.DisableTransportSecurity {
		log.Warn("gRPC encryption is disabled. Disabling transport security...")
		return insecure.NewCredentials(), nil
	}

	var (
		certPool *x509.CertPool
		err      error
	)
	if cfg.TLS.UseSystemCertPool {
		certPool, err = x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("while getting system certificate pool: %w", err)
		}
	} else {
		certPool = x509.NewCertPool()
	}

	if len(cfg.TLS.CACertificate) != 0 {
		log.Debug("Adding custom CA certificate for gRPC connection")
		if !certPool.AppendCertsFromPEM(cfg.TLS.CACertificate) {
			return nil, fmt.Errorf("failed to append CA certificate for gRPC connection")
		}
	}

	if cfg.TLS.InsecureSkipVerify {
		log.Warn("InsecureSkipVerify is enabled. Skipping TLS certificate verification...")
	}

	//nolint:gosec // G402: TLS InsecureSkipVerify may be true. - Yes, indeed - just for development purposes.
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13, InsecureSkipVerify: cfg.TLS.InsecureSkipVerify, RootCAs: certPool}
	return credentials.NewTLS(tlsCfg), nil
}
