package moexoptcalc

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"time"
)

//go:embed certs/russian_trusted_root_ca.pem
var russianTrustedRootCA []byte

// defaultHTTPClient creates an HTTP client that trusts the MOEX certificate chain.
func defaultHTTPClient(timeout time.Duration) (*http.Client, error) {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		// A process may install a custom RoundTripper globally. It owns its
		// transport policy, so preserve it rather than rejecting construction.
		transport := http.DefaultTransport
		if transport == nil {
			transport = &http.Transport{}
		}
		return &http.Client{Transport: transport, Timeout: timeout}, nil
	}
	if base == nil {
		base = &http.Transport{}
	}
	var roots *x509.CertPool
	if base.TLSClientConfig != nil && base.TLSClientConfig.RootCAs != nil {
		roots = base.TLSClientConfig.RootCAs.Clone()
	} else {
		var err error
		roots, err = x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("load system roots: %w", err)
		}
	}
	if roots == nil {
		roots = x509.NewCertPool()
	}
	if !roots.AppendCertsFromPEM(russianTrustedRootCA) {
		return nil, errors.New("invalid embedded Russian Trusted Root CA")
	}
	tr := base.Clone()
	tlsConfig := &tls.Config{}
	if base.TLSClientConfig != nil {
		tlsConfig = base.TLSClientConfig.Clone()
	}
	tlsConfig.RootCAs = roots
	if tlsConfig.MinVersion < tls.VersionTLS12 {
		tlsConfig.MinVersion = tls.VersionTLS12
	}
	tr.TLSClientConfig = tlsConfig
	return &http.Client{Transport: tr, Timeout: timeout}, nil
}
