package moexoptcalc_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/acidsailor/moexoptcalc"
)

// newTestClient builds a client against the public MOEX endpoint. The default
// 30s timeout absorbs occasional ISS sluggishness; t.Run blocks still honour
// t.Deadline via the request context.
func newTestClient(t *testing.T) *moexoptcalc.Client {
	t.Helper()
	c, err := moexoptcalc.NewClient(moexoptcalc.EndpointProduction)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// TestNew_InvalidConfig covers construction-time validation: errors must
// unwrap to *ConfigError.
func TestNew_InvalidConfig(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		endpoint string
	}{
		{"empty endpoint", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := moexoptcalc.NewClient(tc.endpoint)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			var cfgErr *moexoptcalc.ConfigError
			if !errors.As(err, &cfgErr) {
				t.Errorf("err %v should unwrap to *ConfigError", err)
			}
		})
	}
}

// TestWithHTTPClient_StubIsCalled verifies WithHTTPClient installs a custom
// *http.Client whose RoundTripper intercepts outgoing requests.
func TestWithHTTPClient_StubIsCalled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[]"))
		},
	))
	defer srv.Close()

	var called atomic.Int32
	stub := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		called.Add(1)
		return http.DefaultTransport.RoundTrip(req)
	})

	c, err := moexoptcalc.NewClient(
		srv.URL,
		moexoptcalc.WithHTTPClient(
			&http.Client{Transport: stub, Timeout: 30 * time.Second},
		),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.ListAssets(context.Background(), moexoptcalc.ListAssetsRequest{})
	if err != nil {
		t.Fatalf("ListAssets: %v", err)
	}
	if got := called.Load(); got != 1 {
		t.Errorf("stub RoundTripper calls: got %d, want 1", got)
	}
}

// TestWithHTTPClient_NilFallsBackToDefault verifies a nil *http.Client option
// is benign: New ignores it and uses restkit's default.
func TestWithHTTPClient_NilFallsBackToDefault(t *testing.T) {
	t.Parallel()
	c, err := moexoptcalc.NewClient(moexoptcalc.EndpointProduction,
		moexoptcalc.WithHTTPClient(nil))
	if err != nil {
		t.Fatalf(
			"nil *http.Client should fall back to the default, got: %v",
			err,
		)
	}
	if c == nil {
		t.Fatal("expected a non-nil client")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// TestListAssets_Live hits /assets with no filter and verifies the list is
// non-empty and contains a known underlying (Si — USD/RUB futures). Skipped
// under -short.
func TestListAssets_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("live API call")
	}
	t.Parallel()
	c := newTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	assets, err := c.ListAssets(ctx, moexoptcalc.ListAssetsRequest{})
	if err != nil {
		t.Fatalf("ListAssets: %v", err)
	}
	if len(assets) == 0 {
		t.Fatal("ListAssets returned empty list")
	}
	// Unfiltered /assets returns uppercase codes ("SI"); the name=Si
	// filter returns the mixed-case form ("Si"). Accept either spelling.
	foundSi := false
	for _, a := range assets {
		if a.AssetCode == "Si" || a.AssetCode == "SI" {
			foundSi = true
			break
		}
	}
	if !foundSi {
		t.Errorf(
			"expected to find Si/SI among %d assets, codes: %v",
			len(assets),
			assetCodes(assets),
		)
	}
}

func assetCodes(assets []moexoptcalc.Asset) []string {
	out := make([]string, 0, len(assets))
	for _, a := range assets {
		out = append(out, a.AssetCode)
	}
	return out
}

// TestGetOption_Live hits the single-option endpoint and verifies every
// required field decodes. The secid is discovered dynamically via
// ListSeriesOptions on the nearest Si series so the test stays green as
// options roll off.
func TestGetOption_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("live API call")
	}
	t.Parallel()
	c := newTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	series, err := c.ListOptionSeries(ctx, moexoptcalc.ListOptionSeriesRequest{
		AssetCode: "Si",
	})
	if err != nil {
		t.Fatalf("ListOptionSeries(Si): %v", err)
	}
	if len(series) == 0 {
		t.Fatal("ListOptionSeries(Si) returned empty list")
	}
	seriesCode := series[0].OptionseriesCode
	opts, err := c.ListSeriesOptions(ctx, moexoptcalc.ListSeriesOptionsRequest{
		AssetCode:        "Si",
		OptionseriesCode: seriesCode,
	})
	if err != nil {
		t.Fatalf("ListSeriesOptions: %v", err)
	}
	if len(opts) == 0 {
		t.Fatalf("series %s has no options", seriesCode)
	}
	secid := opts[0].Secid

	brief, err := c.GetOption(ctx, moexoptcalc.GetOptionRequest{
		AssetCode: "Si",
		Secid:     secid,
	})
	if err != nil {
		t.Fatalf("GetOption(Si, %s): %v", secid, err)
	}
	if brief.Secid != secid {
		t.Errorf("Secid: got %q, want %q", brief.Secid, secid)
	}
	if brief.UnderlyingAsset == "" {
		t.Error("UnderlyingAsset is empty")
	}
	if brief.ExpiringDate.IsZero() {
		t.Error("ExpiringDate is empty")
	}
	if brief.DaysUntilExpiring < 0 {
		t.Errorf("DaysUntilExpiring negative: %d", brief.DaysUntilExpiring)
	}
}

// TestGetOption_NotFound verifies the non-2xx error contract: the error must
// unwrap to *moexoptcalc.ResponseError carrying the 4xx status code.
func TestGetOption_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("live API call")
	}
	t.Parallel()
	c := newTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_, err := c.GetOption(ctx, moexoptcalc.GetOptionRequest{
		AssetCode: "Si",
		Secid:     "NOTAREALSEC",
	})
	if err == nil {
		t.Fatal("expected error for bogus secid, got nil")
	}
	var apiErr *moexoptcalc.ResponseError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err %v should unwrap to *moexoptcalc.ResponseError", err)
	}
	if apiErr.StatusCode < 400 || apiErr.StatusCode >= 500 {
		t.Errorf("StatusCode: got %d, want a 4xx", apiErr.StatusCode)
	}
}

// TestGetAsset_AmbiguousCode verifies the 422 contract for an unqualified code
// matching multiple asset types (LKOH = share + futures basis). MOEX returns
// `{"detail":"<...asset_type...>"}`; the transport carries the raw body
// verbatim, so the 422 survives as a *moexoptcalc.ResponseError with its body
// intact.
func TestGetAsset_AmbiguousCode(t *testing.T) {
	if testing.Short() {
		t.Skip("live API call")
	}
	t.Parallel()
	c := newTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_, err := c.GetAsset(ctx, moexoptcalc.GetAssetRequest{AssetCode: "LKOH"})
	if err == nil {
		t.Fatal("expected 422 for ambiguous code LKOH, got nil")
	}
	var apiErr *moexoptcalc.ResponseError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err %v should unwrap to *moexoptcalc.ResponseError", err)
	}
	if apiErr.StatusCode != 422 {
		t.Errorf("StatusCode: got %d, want 422", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Body, "asset_type") {
		t.Errorf(
			"Body %q should carry the MOEX detail mentioning asset_type",
			apiErr.Body,
		)
	}
}
