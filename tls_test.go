package moexoptcalc

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"testing"
	"time"
)

func TestDefaultHTTPClientPreservesMOEXTrustWithTimeout(t *testing.T) {
	t.Parallel()
	block, _ := pem.Decode(russianTrustedRootCA)
	if block == nil {
		t.Fatal("invalid embedded certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	for _, timeout := range []time.Duration{0, 2 * time.Second} {
		client, err := defaultHTTPClient(timeout)
		if err != nil {
			t.Fatal(err)
		}
		if client.Timeout != timeout {
			t.Fatalf("timeout = %s, want %s", client.Timeout, timeout)
		}
		transport, ok := client.Transport.(*http.Transport)
		if !ok || transport.TLSClientConfig == nil {
			t.Fatal("default client must use a TLS-aware HTTP transport")
		}
		defer transport.CloseIdleConnections()
		cfg := transport.TLSClientConfig
		if cfg.InsecureSkipVerify || cfg.MinVersion < tls.VersionTLS12 {
			t.Fatal("default client must verify certificates with TLS 1.2+")
		}
		_, err = cert.Verify(x509.VerifyOptions{
			Roots:       cfg.RootCAs,
			CurrentTime: cert.NotBefore.Add(time.Hour),
		})
		if err != nil {
			t.Fatalf("embedded MOEX root not trusted: %v", err)
		}
	}
}
