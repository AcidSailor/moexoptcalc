package moexoptcalc

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"testing"
	"time"
)

func TestEmbeddedRootCA(t *testing.T) {
	block, _ := pem.Decode(russianTrustedRootCA)
	if block == nil {
		t.Fatal("invalid embedded certificate")
	}
	const want = "d26d2d0231b7c39f92cc738512ba54103519e4405d68b5bd703e9788ca8ecf31"
	sum := sha256.Sum256(block.Bytes)
	if got := hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("embedded root fingerprint = %s, want %s", got, want)
	}
}

func TestDefaultClientsOwnTheirTransports(t *testing.T) {
	first, err := defaultHTTPClient(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	second, err := defaultHTTPClient(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if first.Transport == second.Transport || first.Transport == nil ||
		second.Transport == nil {
		t.Fatal("each client must own an independent HTTP transport")
	}
}
