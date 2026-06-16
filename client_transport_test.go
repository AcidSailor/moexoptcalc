package moexoptcalc_test

import (
	"context"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/acidsailor/moexoptcalc"
)

// TestListOptions_RequestBuilding checks path + query assembly for a GET with
// mixed optional filters, and that a JSON array body decodes.
func TestListOptions_RequestBuilding(t *testing.T) {
	t.Parallel()
	var gotPath, gotQuery string
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, "[]")
		}),
	)
	defer srv.Close()

	c, err := moexoptcalc.New(srv.URL)
	require.NoError(t, err, "New")
	at := "futures"
	ot := "call"
	got, err := c.ListOptions(
		context.Background(),
		moexoptcalc.ListOptionsParams{
			AssetCode:  "Si",
			AssetType:  &at,
			OptionType: &ot,
		},
	)
	require.NoError(t, err, "ListOptions")
	assert.Empty(t, got, "expected empty slice")
	assert.Equal(t, "/assets/Si/options", gotPath, "path")
	assert.Equal(t, "asset_type=futures&option_type=call", gotQuery, "query")
}

// TestCalculateInitialMargin_PostBody checks the POST body is the bare JSON
// array and a scalar response decodes. Generated InitialMarginPosition fields
// emit in struct (alphabetical) order — price, quantity, secid — with the
// optional netted_im omitted.
func TestCalculateInitialMargin_PostBody(t *testing.T) {
	t.Parallel()
	var gotBody, gotCT string
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			gotCT = r.Header.Get("Content-Type")
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, "1234.5")
		}),
	)
	defer srv.Close()

	c, err := moexoptcalc.New(srv.URL)
	require.NoError(t, err, "New")
	got, err := c.CalculateInitialMargin(
		context.Background(),
		[]moexoptcalc.InitialMarginPosition{
			{Secid: "Si-1", Quantity: 2, Price: 1000.0},
		},
	)
	require.NoError(t, err, "CalculateInitialMargin")
	assert.Equal(t, 1234.5, got, "margin")
	assert.Equal(t, "application/json", gotCT, "content-type")
	assert.Equal(
		t,
		`[{"price":1000,"quantity":2,"secid":"Si-1"}]`,
		gotBody,
		"body",
	)
}

// captureServer returns an httptest server that records the last request's
// path and raw query and replies with an empty JSON array, plus pointers to
// read them back after the call.
func captureServer(t *testing.T) (*httptest.Server, *string, *string) {
	t.Helper()
	var gotPath, gotQuery string
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, "[]")
		}),
	)
	t.Cleanup(srv.Close)
	return srv, &gotPath, &gotQuery
}

// TestListFutures_DateQuery checks a non-nil *Date filter reaches the wire in
// the bare "2006-01-02" form under expiration_date — the hand-written setDate
// path the generic restkit setters can't cover.
func TestListFutures_DateQuery(t *testing.T) {
	t.Parallel()
	srv, gotPath, gotQuery := captureServer(t)

	c, err := moexoptcalc.New(srv.URL)
	require.NoError(t, err, "New")
	exp := moexoptcalc.NewDate(time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC))
	_, err = c.ListFutures(context.Background(), moexoptcalc.ListFuturesParams{
		AssetCode:      "Si",
		ExpirationDate: &exp,
	})
	require.NoError(t, err, "ListFutures")
	assert.Equal(t, "/assets/Si/futures", *gotPath, "path")
	assert.Equal(t, "expiration_date=2026-05-12", *gotQuery, "query")
}

// TestListFutures_NilDateOmitted checks a nil *Date filter is dropped rather
// than sent as an empty value.
func TestListFutures_NilDateOmitted(t *testing.T) {
	t.Parallel()
	srv, _, gotQuery := captureServer(t)

	c, err := moexoptcalc.New(srv.URL)
	require.NoError(t, err, "New")
	_, err = c.ListFutures(context.Background(), moexoptcalc.ListFuturesParams{
		AssetCode: "Si",
	})
	require.NoError(t, err, "ListFutures")
	assert.Empty(t, *gotQuery, "nil date must be omitted from the query")
}

// TestStrike_WireTypes locks the deliberate per-endpoint strike type split:
// ListOptions sends a float (spec: number), ListSeriesOptions an integer
// (spec: integer), both under the same "strike" key.
func TestStrike_WireTypes(t *testing.T) {
	t.Parallel()

	t.Run("ListOptions sends float", func(t *testing.T) {
		t.Parallel()
		srv, _, gotQuery := captureServer(t)
		c, err := moexoptcalc.New(srv.URL)
		require.NoError(t, err, "New")
		strike := 95000.5
		_, err = c.ListOptions(
			context.Background(),
			moexoptcalc.ListOptionsParams{
				AssetCode: "Si",
				Strike:    &strike,
			},
		)
		require.NoError(t, err, "ListOptions")
		assert.Equal(t, "strike=95000.5", *gotQuery, "float strike")
	})

	t.Run("ListSeriesOptions sends integer", func(t *testing.T) {
		t.Parallel()
		srv, _, gotQuery := captureServer(t)
		c, err := moexoptcalc.New(srv.URL)
		require.NoError(t, err, "New")
		strike := int32(95000)
		_, err = c.ListSeriesOptions(
			context.Background(),
			moexoptcalc.ListSeriesOptionsParams{
				AssetCode:        "Si",
				OptionseriesCode: "Si-6.26",
				Strike:           &strike,
			},
		)
		require.NoError(t, err, "ListSeriesOptions")
		assert.Equal(t, "strike=95000", *gotQuery, "integer strike")
	})
}

// TestPathEscaping verifies path segments are url.PathEscape'd: a code with a
// "?" stays in the path (escaped) rather than splitting off a query string.
func TestPathEscaping(t *testing.T) {
	t.Parallel()
	srv, gotPath, gotQuery := captureServer(t)

	c, err := moexoptcalc.New(srv.URL)
	require.NoError(t, err, "New")
	_, err = c.ListOptions(context.Background(), moexoptcalc.ListOptionsParams{
		AssetCode: "Si?x",
	})
	require.NoError(t, err, "ListOptions")
	assert.Equal(
		t,
		"/assets/Si?x/options",
		*gotPath,
		"path keeps the escaped code",
	)
	assert.Empty(t, *gotQuery, "the '?' must not leak into a query string")
}

// TestErrorMapping verifies a non-2xx response maps to *ResponseError carrying
// the status and raw body.
func TestErrorMapping(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = io.WriteString(w, `{"detail":"pick an asset_type"}`)
		}),
	)
	defer srv.Close()

	c, err := moexoptcalc.New(srv.URL)
	require.NoError(t, err, "New")
	_, err = c.GetAsset(
		context.Background(),
		moexoptcalc.GetAssetParams{AssetCode: "LKOH"},
	)
	require.Error(t, err, "expected error")
	var respErr *moexoptcalc.ResponseError
	require.ErrorAs(t, err, &respErr, "err should unwrap to *ResponseError")
	assert.Equal(t, 422, respErr.StatusCode, "StatusCode")
	assert.Equal(t, `{"detail":"pick an asset_type"}`, respErr.Body, "Body")

	// A non-2xx is *ResponseError only — not a *RequestError.
	var reqErr *moexoptcalc.RequestError
	assert.NotErrorAs(t, err, &reqErr, "not a RequestError")
}

// TestDecodeError verifies a 2xx body that isn't valid JSON for the target
// type surfaces as *RequestError with Op "unmarshal", not a *ResponseError.
func TestDecodeError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(
				w,
				`{"not":"an array"}`,
			) // ListAssets wants []Asset
		}),
	)
	defer srv.Close()

	c, err := moexoptcalc.New(srv.URL)
	require.NoError(t, err, "New")
	_, err = c.ListAssets(context.Background(), moexoptcalc.ListAssetsParams{})
	require.Error(t, err, "expected decode error")
	var reqErr *moexoptcalc.RequestError
	require.ErrorAs(t, err, &reqErr, "decode failure should be a *RequestError")
	assert.Equal(t, moexoptcalc.OpUnmarshal, reqErr.Op, "Op")

	var respErr *moexoptcalc.ResponseError
	assert.NotErrorAs(
		t,
		err,
		&respErr,
		"decode failure is not a *ResponseError",
	)
}

// TestEncodeError verifies a request body that can't be JSON-encoded (a NaN
// float) surfaces as *RequestError with Op "marshal" and never reaches the
// wire.
func TestEncodeError(t *testing.T) {
	t.Parallel()
	var hit bool
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(http.ResponseWriter, *http.Request) { hit = true },
		),
	)
	defer srv.Close()

	c, err := moexoptcalc.New(srv.URL)
	require.NoError(t, err, "New")
	_, err = c.CalculateInitialMargin(
		context.Background(),
		[]moexoptcalc.InitialMarginPosition{{Secid: "Si-1", Price: math.NaN()}},
	)
	require.Error(t, err, "expected encode error")
	var reqErr *moexoptcalc.RequestError
	require.ErrorAs(t, err, &reqErr, "encode failure should be a *RequestError")
	assert.Equal(t, moexoptcalc.OpMarshal, reqErr.Op, "Op")
	assert.False(t, hit, "request must not reach the server")
}

// TestTransportError verifies a cancelled context surfaces as *RequestError
// with Op "send", preserving the wrapped context.Canceled cause.
func TestTransportError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, "[]")
		}),
	)
	defer srv.Close()

	c, err := moexoptcalc.New(srv.URL)
	require.NoError(t, err, "New")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call so Do fails immediately
	_, err = c.ListAssets(ctx, moexoptcalc.ListAssetsParams{})
	require.Error(t, err, "expected transport error")
	var reqErr *moexoptcalc.RequestError
	require.ErrorAs(
		t,
		err,
		&reqErr,
		"transport failure should be a *RequestError",
	)
	assert.Equal(t, "send", reqErr.Op, "Op")
	assert.ErrorIs(t, err, context.Canceled, "wrapped cause should survive")
}
