// Package moexoptcalc is a Go client for the public MOEX ISS Options
// Calculator API. Models are oapi-codegen output (models.gen.go); the HTTP
// transport is the shared restkit core. The API is public — no auth required.
package moexoptcalc

import (
	"context"
	"net/http"
	"net/url"

	"github.com/acidsailor/restkit"
)

const (
	// EndpointProduction is the public MOEX ISS server (no test environment).
	EndpointProduction = "https://iss.moex.com/iss/apps/option-calc/v1"

	clientName = "moexoptcalc"
)

// Wire (snake_case) query-parameter keys, mirroring the json tags on the
// *Params structs in params.go.
const (
	keyAssetType         = "asset_type"
	keyAssetSubtype      = "asset_subtype"
	keyQuery             = "query"
	keyExpirationDate    = "expiration_date"
	keyOptionType        = "option_type"
	keySeriesType        = "series_type"
	keyStrike            = "strike"
	keyDaysUntilExpiring = "days_until_expiring"
	keyUnderlyingPrice   = "underlying_price"
	keyVolatility        = "volatility"
	keyRows              = "rows"
)

// Client is the MOEX Options Calculator API client. It is immutable after
// construction and safe for concurrent use.
type Client struct {
	rkClient *restkit.Client
}

// ClientOption configures a Client at construction time. See WithHTTPClient.
// (Named ClientOption, not Option, since Option is the generated model for an
// option instrument.)
type ClientOption func(*config)

type config struct {
	httpClient *http.Client
}

// WithHTTPClient sets the *http.Client for outgoing API calls — for a custom
// Timeout, Transport, proxy, etc. A nil client is ignored: New falls back to
// restkit's default (30s Timeout, stdlib default Transport).
func WithHTTPClient(h *http.Client) ClientOption {
	return func(c *config) { c.httpClient = h }
}

// New builds a Client targeting endpoint, applying any options. Use
// EndpointProduction for the public MOEX ISS host. Returns *ConfigError on an
// empty endpoint.
func New(endpoint string, opts ...ClientOption) (*Client, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}
	// A nil cfg.httpClient is benign: restkit falls back to its default.
	rkClient, err := restkit.New(
		endpoint,
		restkit.WithName(clientName),
		restkit.WithHTTPClient(cfg.httpClient),
	)
	if err != nil {
		return nil, err
	}
	return &Client{rkClient: rkClient}, nil
}

// setDate adds a non-nil *Date filter to v in the wire date format. The
// generic restkit.Values setters can't carry the package-local Date type, so
// dates go through the embedded Set behind this nil-check.
func setDate(v restkit.Values, key string, d *Date) restkit.Values {
	if d != nil {
		v.Set(key, d.Format(dateLayout))
	}
	return v
}

// ListAssets returns underlying assets, optionally filtered. GET /assets.
func (c *Client) ListAssets(
	ctx context.Context,
	params ListAssetsParams,
) ([]Asset, error) {
	q := restkit.NewValues().
		Str(keyAssetType, params.AssetType).
		Str(keyAssetSubtype, params.AssetSubtype).
		Str(keyQuery, params.Query)
	return restkit.Do[[]Asset](
		ctx,
		c.rkClient,
		http.MethodGet,
		"/assets",
		nil,
		restkit.WithQuery(q.Values),
	)
}

// GetAsset returns a single underlying. GET /assets/{asset_code}.
func (c *Client) GetAsset(
	ctx context.Context,
	params GetAssetParams,
) (Asset, error) {
	q := restkit.NewValues().Str(keyAssetType, params.AssetType)
	return restkit.Do[Asset](
		ctx,
		c.rkClient,
		http.MethodGet,
		"/assets/"+url.PathEscape(
			params.AssetCode,
		),
		nil,
		restkit.WithQuery(q.Values),
	)
}

// ListFutures returns futures on an underlying. GET /assets/{asset_code}/futures.
func (c *Client) ListFutures(
	ctx context.Context,
	params ListFuturesParams,
) ([]Futures, error) {
	q := setDate(restkit.NewValues(), keyExpirationDate, params.ExpirationDate)
	return restkit.Do[[]Futures](
		ctx,
		c.rkClient,
		http.MethodGet,
		"/assets/"+url.PathEscape(
			params.AssetCode,
		)+"/futures",
		nil,
		restkit.WithQuery(q.Values),
	)
}

// ListOptions returns options on an underlying. GET /assets/{asset_code}/options.
func (c *Client) ListOptions(
	ctx context.Context,
	params ListOptionsParams,
) ([]Option, error) {
	q := restkit.NewValues().
		Str(keyAssetType, params.AssetType).
		Str(keyOptionType, params.OptionType).
		Str(keySeriesType, params.SeriesType).
		Float(keyStrike, params.Strike)
	q = setDate(q, keyExpirationDate, params.ExpirationDate)
	return restkit.Do[[]Option](
		ctx,
		c.rkClient,
		http.MethodGet,
		"/assets/"+url.PathEscape(
			params.AssetCode,
		)+"/options",
		nil,
		restkit.WithQuery(q.Values),
	)
}

// GetOption returns greeks/IV for one option. GET /assets/{asset_code}/options/{secid}.
func (c *Client) GetOption(
	ctx context.Context,
	params GetOptionParams,
) (OptionBrief, error) {
	q := restkit.NewValues().
		Str(keyAssetType, params.AssetType).
		Int32(keyDaysUntilExpiring, params.DaysUntilExpiring).
		Float(keyUnderlyingPrice, params.UnderlyingPrice).
		Float(keyVolatility, params.Volatility)
	return restkit.Do[OptionBrief](
		ctx,
		c.rkClient,
		http.MethodGet,
		"/assets/"+url.PathEscape(
			params.AssetCode,
		)+"/options/"+url.PathEscape(
			params.Secid,
		),
		nil,
		restkit.WithQuery(q.Values),
	)
}

// ListOptionSeries returns series on an underlying. GET /assets/{asset_code}/optionseries.
func (c *Client) ListOptionSeries(
	ctx context.Context,
	params ListOptionSeriesParams,
) ([]OptionSeries, error) {
	q := restkit.NewValues().Str(keyAssetType, params.AssetType)
	return restkit.Do[[]OptionSeries](
		ctx,
		c.rkClient,
		http.MethodGet,
		"/assets/"+url.PathEscape(
			params.AssetCode,
		)+"/optionseries",
		nil,
		restkit.WithQuery(q.Values),
	)
}

// GetOptionSeries returns one series. GET /assets/{asset_code}/optionseries/{optionseries_code}.
func (c *Client) GetOptionSeries(
	ctx context.Context,
	params GetOptionSeriesParams,
) (OptionSeries, error) {
	q := restkit.NewValues().Str(keyAssetType, params.AssetType)
	return restkit.Do[OptionSeries](
		ctx,
		c.rkClient,
		http.MethodGet,
		"/assets/"+url.PathEscape(
			params.AssetCode,
		)+"/optionseries/"+url.PathEscape(
			params.OptionseriesCode,
		),
		nil,
		restkit.WithQuery(q.Values),
	)
}

// ListSeriesOptions returns options in a series. GET .../optionseries/{optionseries_code}/options.
func (c *Client) ListSeriesOptions(
	ctx context.Context,
	params ListSeriesOptionsParams,
) ([]Option, error) {
	q := restkit.NewValues().
		Str(keyAssetType, params.AssetType).
		Str(keyOptionType, params.OptionType).
		Int32(keyStrike, params.Strike)
	return restkit.Do[[]Option](
		ctx,
		c.rkClient,
		http.MethodGet,
		"/assets/"+url.PathEscape(
			params.AssetCode,
		)+"/optionseries/"+url.PathEscape(
			params.OptionseriesCode,
		)+"/options",
		nil,
		restkit.WithQuery(q.Values),
	)
}

// GetOptionBoard returns the strike×type board. GET .../optionseries/{optionseries_code}/optionboard.
func (c *Client) GetOptionBoard(
	ctx context.Context,
	params GetOptionBoardParams,
) (OptionBoard, error) {
	q := restkit.NewValues().
		Str(keyAssetType, params.AssetType).
		Int32(keyRows, params.Rows)
	return restkit.Do[OptionBoard](
		ctx,
		c.rkClient,
		http.MethodGet,
		"/assets/"+url.PathEscape(
			params.AssetCode,
		)+"/optionseries/"+url.PathEscape(
			params.OptionseriesCode,
		)+"/optionboard",
		nil,
		restkit.WithQuery(q.Values),
	)
}

// GetVolatilityGraph returns IV smile points. GET .../optionseries/{optionseries_code}/volatility_graph.
func (c *Client) GetVolatilityGraph(
	ctx context.Context,
	params GetVolatilityGraphParams,
) ([]VolatilityGraphPoint, error) {
	q := restkit.NewValues().Str(keyAssetType, params.AssetType)
	return restkit.Do[[]VolatilityGraphPoint](
		ctx,
		c.rkClient,
		http.MethodGet,
		"/assets/"+url.PathEscape(
			params.AssetCode,
		)+"/optionseries/"+url.PathEscape(
			params.OptionseriesCode,
		)+"/volatility_graph",
		nil,
		restkit.WithQuery(q.Values),
	)
}

// CalculatePortfolio returns portfolio greeks and P&L. POST /portfolio/.
func (c *Client) CalculatePortfolio(
	ctx context.Context,
	req OptionPortfolio,
) (CalculatedPortfolio, error) {
	return restkit.Do[CalculatedPortfolio](
		ctx,
		c.rkClient,
		http.MethodPost,
		"/portfolio/",
		req,
	)
}

// CalculatePortfolioGraph returns a scenario graph. POST /portfolio/graph/{indicator}.
func (c *Client) CalculatePortfolioGraph(
	ctx context.Context,
	req OptionPortfolio,
	params CalculatePortfolioGraphParams,
) (IndicatorGraph, error) {
	return restkit.Do[IndicatorGraph](
		ctx,
		c.rkClient,
		http.MethodPost,
		"/portfolio/graph/"+url.PathEscape(params.Indicator),
		req,
	)
}

// CalculateInitialMargin returns the initial margin. POST /portfolio/initial_margin.
func (c *Client) CalculateInitialMargin(
	ctx context.Context,
	req []InitialMarginPosition,
) (float64, error) {
	return restkit.Do[float64](
		ctx,
		c.rkClient,
		http.MethodPost,
		"/portfolio/initial_margin",
		req,
	)
}
