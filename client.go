// Package moexoptcalc is a Go client for the public MOEX ISS Options
// Calculator API. Models are oapi-codegen output (models.gen.go); the HTTP
// transport is the shared restkit core. The API is public — no auth required.
package moexoptcalc

import (
	"context"
	"net/http"

	"github.com/acidsailor/restkit"
)

const (
	// EndpointProduction is the public MOEX ISS server (no test environment).
	EndpointProduction = "https://iss.moex.com/iss/apps/option-calc/v1"

	clientName = "moexoptcalc"
)

// Wire (snake_case) query-parameter keys, mirroring the json tags on the
// *Request structs in params.go.
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
// Timeout, Transport, proxy, etc. A nil client is ignored: NewClient falls back to
// restkit's default (30s Timeout, stdlib default Transport).
func WithHTTPClient(h *http.Client) ClientOption {
	return func(c *config) { c.httpClient = h }
}

// NewClient builds a Client targeting endpoint, applying any options. Use
// EndpointProduction for the public MOEX ISS host. Returns *ConfigError on an
// empty endpoint.
func NewClient(endpoint string, opts ...ClientOption) (*Client, error) {
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

// get issues a GET for path with query q and decodes the JSON body into T —
// the shape shared by every read endpoint.
func get[T any](
	ctx context.Context,
	c *Client,
	path string,
	q restkit.Values,
) (T, error) {
	return restkit.Do[T](
		ctx,
		c.rkClient,
		http.MethodGet,
		path,
		nil,
		restkit.WithQuery(q.Values),
	)
}

// ListAssets returns underlying assets, optionally filtered. GET /assets.
func (c *Client) ListAssets(
	ctx context.Context,
	params ListAssetsRequest,
) ([]Asset, error) {
	q := restkit.NewValues().
		Str(keyAssetType, params.AssetType).
		Str(keyAssetSubtype, params.AssetSubtype).
		Str(keyQuery, params.Query)
	return get[[]Asset](ctx, c, "/assets", q)
}

// GetAsset returns a single underlying. GET /assets/{asset_code}.
func (c *Client) GetAsset(
	ctx context.Context,
	params GetAssetRequest,
) (*Asset, error) {
	q := restkit.NewValues().Str(keyAssetType, params.AssetType)
	return get[*Asset](ctx, c, restkit.Pathf("/assets/%s", params.AssetCode), q)
}

// ListFutures returns futures on an underlying. GET /assets/{asset_code}/futures.
func (c *Client) ListFutures(
	ctx context.Context,
	params ListFuturesRequest,
) ([]Futures, error) {
	q := restkit.NewValues().Param(keyExpirationDate, params.ExpirationDate)
	return get[[]Futures](
		ctx,
		c,
		restkit.Pathf("/assets/%s/futures", params.AssetCode),
		q,
	)
}

// ListOptions returns options on an underlying. GET /assets/{asset_code}/options.
func (c *Client) ListOptions(
	ctx context.Context,
	params ListOptionsRequest,
) ([]Option, error) {
	q := restkit.NewValues().
		Str(keyAssetType, params.AssetType).
		Str(keyOptionType, params.OptionType).
		Str(keySeriesType, params.SeriesType).
		Float(keyStrike, params.Strike).
		Param(keyExpirationDate, params.ExpirationDate)
	return get[[]Option](
		ctx,
		c,
		restkit.Pathf("/assets/%s/options", params.AssetCode),
		q,
	)
}

// GetOption returns greeks/IV for one option. GET /assets/{asset_code}/options/{secid}.
func (c *Client) GetOption(
	ctx context.Context,
	params GetOptionRequest,
) (*OptionBrief, error) {
	q := restkit.NewValues().
		Str(keyAssetType, params.AssetType).
		Int32(keyDaysUntilExpiring, params.DaysUntilExpiring).
		Float(keyUnderlyingPrice, params.UnderlyingPrice).
		Float(keyVolatility, params.Volatility)
	return get[*OptionBrief](
		ctx,
		c,
		restkit.Pathf(
			"/assets/%s/options/%s",
			params.AssetCode,
			params.Secid,
		),
		q,
	)
}

// ListOptionSeries returns series on an underlying. GET /assets/{asset_code}/optionseries.
func (c *Client) ListOptionSeries(
	ctx context.Context,
	params ListOptionSeriesRequest,
) ([]OptionSeries, error) {
	q := restkit.NewValues().Str(keyAssetType, params.AssetType)
	return get[[]OptionSeries](
		ctx,
		c,
		restkit.Pathf("/assets/%s/optionseries", params.AssetCode),
		q,
	)
}

// GetOptionSeries returns one series. GET /assets/{asset_code}/optionseries/{optionseries_code}.
func (c *Client) GetOptionSeries(
	ctx context.Context,
	params GetOptionSeriesRequest,
) (*OptionSeries, error) {
	q := restkit.NewValues().Str(keyAssetType, params.AssetType)
	return get[*OptionSeries](
		ctx,
		c,
		restkit.Pathf(
			"/assets/%s/optionseries/%s",
			params.AssetCode,
			params.OptionseriesCode,
		),
		q,
	)
}

// ListSeriesOptions returns options in a series. GET .../optionseries/{optionseries_code}/options.
func (c *Client) ListSeriesOptions(
	ctx context.Context,
	params ListSeriesOptionsRequest,
) ([]Option, error) {
	q := restkit.NewValues().
		Str(keyAssetType, params.AssetType).
		Str(keyOptionType, params.OptionType).
		Int32(keyStrike, params.Strike)
	return get[[]Option](
		ctx,
		c,
		restkit.Pathf(
			"/assets/%s/optionseries/%s/options",
			params.AssetCode,
			params.OptionseriesCode,
		),
		q,
	)
}

// GetOptionBoard returns the strike×type board. GET .../optionseries/{optionseries_code}/optionboard.
func (c *Client) GetOptionBoard(
	ctx context.Context,
	params GetOptionBoardRequest,
) (*OptionBoard, error) {
	q := restkit.NewValues().
		Str(keyAssetType, params.AssetType).
		Int32(keyRows, params.Rows)
	return get[*OptionBoard](
		ctx,
		c,
		restkit.Pathf(
			"/assets/%s/optionseries/%s/optionboard",
			params.AssetCode,
			params.OptionseriesCode,
		),
		q,
	)
}

// GetVolatilityGraph returns IV smile points. GET .../optionseries/{optionseries_code}/volatility_graph.
func (c *Client) GetVolatilityGraph(
	ctx context.Context,
	params GetVolatilityGraphRequest,
) ([]VolatilityGraphPoint, error) {
	q := restkit.NewValues().Str(keyAssetType, params.AssetType)
	return get[[]VolatilityGraphPoint](
		ctx,
		c,
		restkit.Pathf(
			"/assets/%s/optionseries/%s/volatility_graph",
			params.AssetCode,
			params.OptionseriesCode,
		),
		q,
	)
}

// CalculatePortfolio returns portfolio greeks and P&L. POST /portfolio/.
func (c *Client) CalculatePortfolio(
	ctx context.Context,
	req OptionPortfolio,
) (*CalculatedPortfolio, error) {
	return restkit.Do[*CalculatedPortfolio](
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
	params CalculatePortfolioGraphRequest,
) (*IndicatorGraph, error) {
	return restkit.Do[*IndicatorGraph](
		ctx,
		c.rkClient,
		http.MethodPost,
		restkit.Pathf("/portfolio/graph/%s", params.Indicator),
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
