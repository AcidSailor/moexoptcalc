package moexoptcalc_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/acidsailor/moexoptcalc"
)

func TestClientTimeout(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts []moexoptcalc.ClientOption
		want time.Duration
	}{
		{name: "default", want: 30 * time.Second},
		{
			name: "custom",
			opts: []moexoptcalc.ClientOption{
				moexoptcalc.WithTimeout(2 * time.Second),
			},
			want: 2 * time.Second,
		},
		{
			name: "zero disables timeout",
			opts: []moexoptcalc.ClientOption{moexoptcalc.WithTimeout(0)},
		},
		{
			name: "negative disables timeout",
			opts: []moexoptcalc.ClientOption{moexoptcalc.WithTimeout(-1)},
		},
		{
			name: "last timeout wins",
			opts: []moexoptcalc.ClientOption{
				moexoptcalc.WithTimeout(time.Second),
				moexoptcalc.WithTimeout(2 * time.Second),
			},
			want: 2 * time.Second,
		},
		{
			name: "nil HTTP client preserves timeout",
			opts: []moexoptcalc.ClientOption{
				moexoptcalc.WithTimeout(2 * time.Second),
				moexoptcalc.WithHTTPClient(nil),
				nil,
			},
			want: 2 * time.Second,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// These subtests must stay sequential: they replace the global
			// transport to inspect the default HTTP client's request deadline.
			original := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = original })
			http.DefaultTransport = timeoutTransport(t, tc.want)
			requestAssets(t, tc.opts...)
		})
	}
}

func TestWithHTTPClientTakesPrecedenceOverTimeout(t *testing.T) {
	t.Parallel()
	for _, timeoutFirst := range []bool{true, false} {
		t.Run(
			map[bool]string{true: "timeout first", false: "client first"}[timeoutFirst],
			func(t *testing.T) {
				t.Parallel()
				h := &http.Client{
					Timeout:   2 * time.Second,
					Transport: timeoutTransport(t, 2*time.Second),
				}
				opts := []moexoptcalc.ClientOption{
					moexoptcalc.WithTimeout(time.Minute),
					moexoptcalc.WithHTTPClient(h),
				}
				if !timeoutFirst {
					opts[0], opts[1] = opts[1], opts[0]
				}
				requestAssets(t, opts...)
				if h.Timeout != 2*time.Second {
					t.Fatalf("caller client timeout mutated: %s", h.Timeout)
				}
			},
		)
	}
}

func timeoutTransport(t *testing.T, want time.Duration) http.RoundTripper {
	t.Helper()
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		deadline, ok := req.Context().Deadline()
		if want <= 0 {
			if ok {
				t.Error("unexpected deadline with timeout disabled")
			}
		} else if !ok {
			t.Error("request deadline missing")
		} else if remaining := time.Until(deadline); remaining <= want-time.Second || remaining > want {
			t.Errorf(
				"remaining deadline = %s, want approximately %s",
				remaining,
				want,
			)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("[]")),
			Request:    req,
		}, nil
	})
}

func requestAssets(t *testing.T, opts ...moexoptcalc.ClientOption) {
	t.Helper()
	client, err := moexoptcalc.NewClient("https://example.invalid", opts...)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListAssets(
		context.Background(),
		moexoptcalc.ListAssetsRequest{},
	); err != nil {
		t.Fatal(err)
	}
}
