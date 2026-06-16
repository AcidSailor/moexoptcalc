# moexoptcalc

Go client for the public **MOEX ISS Options Calculator** API. It returns the
official greeks, implied volatility, theoretical prices, and portfolio/margin
metrics that MOEX publishes for FORTS options — the same numbers used for
margin calculations. The models are generated from MOEX's OpenAPI spec by
[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) as plain Go
structs and wrapped by a hand-written transport over the shared
[restkit](https://github.com/acidsailor/restkit) core. The Options Calculator
API is public — **no authentication is required**.

API docs: <https://iss.moex.com/iss/apps/option-calc/v1/docs>

## Features

- 13 typed methods covering the full option-calc surface: assets, futures,
  options, option series, option board, volatility graph, portfolio
  calculation/graph, and initial margin.
- Plain-struct models (direct field access; optional fields are pointers).
- Date-only (`Date`) and Moscow-zoned date-time (`DateTime`) types with
  correct JSON (un)marshaling — `time/tzdata` is embedded so zones resolve on
  any host.
- Three typed errors matched with `errors.As` (`*ConfigError`,
  `*RequestError`, `*ResponseError`); no sentinel values.
- Built-in OpenTelemetry: every call emits standard `http.client.*` spans and
  metrics through the global OTel providers (opt-in, free until enabled).
- Immutable `Client`, safe for concurrent use.

## Install

```sh
go get github.com/acidsailor/moexoptcalc
```

Requires Go 1.26+.

## Quickstart

```go
package main

import (
	"context"
	"log"

	"github.com/acidsailor/moexoptcalc"
)

func main() {
	c, err := moexoptcalc.New(moexoptcalc.EndpointProduction)
	if err != nil {
		log.Fatal(err)
	}

	opt, err := c.GetOption(context.Background(), moexoptcalc.GetOptionParams{
		AssetCode: "Si",
		Secid:     "SI84500BE6B",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("delta=%v gamma=%v vega=%v theta=%v rho=%v iv=%v theor=%v",
		opt.Delta, opt.Gamma, opt.Vega, opt.Theta, opt.Rho,
		opt.Volatility, opt.Theorprice)
}
```

### Methods

All live on `*moexoptcalc.Client`:

| Method | HTTP | Path |
| --- | --- | --- |
| `ListAssets` | GET | `/assets` |
| `GetAsset` | GET | `/assets/{asset_code}` |
| `ListFutures` | GET | `/assets/{asset_code}/futures` |
| `ListOptions` | GET | `/assets/{asset_code}/options` |
| `GetOption` | GET | `/assets/{asset_code}/options/{secid}` |
| `ListOptionSeries` | GET | `/assets/{asset_code}/optionseries` |
| `GetOptionSeries` | GET | `/assets/{asset_code}/optionseries/{optionseries_code}` |
| `ListSeriesOptions` | GET | `/assets/{asset_code}/optionseries/{optionseries_code}/options` |
| `GetOptionBoard` | GET | `/assets/{asset_code}/optionseries/{optionseries_code}/optionboard` |
| `GetVolatilityGraph` | GET | `/assets/{asset_code}/optionseries/{optionseries_code}/volatility_graph` |
| `CalculatePortfolio` | POST | `/portfolio/` |
| `CalculatePortfolioGraph` | POST | `/portfolio/graph/{indicator}` |
| `CalculateInitialMargin` | POST | `/portfolio/initial_margin` |

## Configuration

`moexoptcalc.New(endpoint, ...ClientOption)` takes the target endpoint —
use `moexoptcalc.EndpointProduction` for the public MOEX ISS host. The
Options Calculator API has no separate test environment and needs no
credentials, so there are **no environment variables** to set.

By default `New` uses an `*http.Client` with a 30s timeout and the stdlib
default transport. Override it with `WithHTTPClient` (for a custom timeout,
transport, or proxy):

```go
c, err := moexoptcalc.New(moexoptcalc.EndpointProduction,
	moexoptcalc.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}))
```

A nil `*http.Client` is ignored, falling back to the default.

### Observability

`New` wraps the HTTP transport with `otelhttp`, so calls emit spans and the
standard HTTP-client metrics (`http.client.request.duration`,
`http.client.request.body.size`) through the **global** OTel providers. Opt
in by registering providers via `otel.SetTracerProvider` /
`otel.SetMeterProvider` before calling `New`; until then the no-op defaults
make it free.

## Errors

No sentinel errors — the typed error *is* the category. The three types are
re-exported from restkit and matched with `errors.As`; wrapped causes survive
an unwrap:

| Type | Returned when |
| --- | --- |
| `*moexoptcalc.ConfigError` | `New` got an empty endpoint. |
| `*moexoptcalc.RequestError` | a call failed to marshal, build, hook, send, read, or unmarshal (incl. a cancelled context). The `Op` field names the stage. |
| `*moexoptcalc.ResponseError` | a non-2xx response. Exposes `StatusCode` and raw `Body`; not a `*RequestError`. |

```go
var respErr *moexoptcalc.ResponseError
if errors.As(err, &respErr) {
	_ = respErr.StatusCode // e.g. 404, 422
	_ = respErr.Body       // e.g. `{"detail":"..."}`
}
if errors.Is(err, context.Canceled) { /* call was cancelled */ }
```

## Regeneration

`models.gen.go` is generated from MOEX's OpenAPI spec by oapi-codegen; the
pipeline runs entirely via Docker (no local codegen toolchain). Do not edit
`models.gen.go` by hand.

```sh
task spec     # download spec/spec-upstream.json from iss.moex.com (committed)
task overlay  # Speakeasy: apply spec/overlay.yml -> spec/spec.json (oapi-codegen input)
task deref    # Redocly bundle --dereferenced -> spec/spec-deref.json (MCP input)
task gen      # overlay + oapi-codegen -> models.gen.go (fails on any warning)
task check    # lint + vet + test
```

## Scope

This module wraps only the option-calc sub-API. Out of scope: WebSocket /
streaming (the API has none), authentication (the API is public), and the
MOEX ISS classic REST API (quotes, candles, board snapshots).

## Disclaimer

This library is provided "as is", without warranty of any kind. The author
assumes **no financial, legal, or other liability** for any losses, damages, or
consequences arising from the use of this library, including but not limited to
losses incurred through trading, order placement, or interaction with the MOEX
ISS Options Calculator API.

Nothing in this library, its documentation, or examples constitutes **investment
advice, a recommendation, or solicitation** to buy or sell any financial
instrument. All trading decisions are solely the responsibility of the user.
Consult a licensed financial advisor before making investment decisions.

## Отказ от ответственности

Библиотека предоставляется «как есть», без каких-либо гарантий. Автор **не несёт
финансовой, юридической или иной ответственности** за любые убытки, ущерб или
последствия, возникшие в результате использования этой библиотеки, включая, но
не ограничиваясь, убытки от торговли, выставления ордеров или взаимодействия с
API MOEX ISS Options Calculator.

Ничто в этой библиотеке, её документации или примерах **не является
индивидуальной инвестиционной рекомендацией**, предложением или побуждением к
покупке или продаже каких-либо финансовых инструментов. Все торговые решения
принимаются пользователем самостоятельно и под его ответственность. Перед
принятием инвестиционных решений проконсультируйтесь с лицензированным
финансовым советником.

## License

GNU Affero General Public License v3.0. See [LICENSE](LICENSE).
