package moexoptcalc

import (
	"encoding/json"
	"fmt"
	"time"
)

// Per-operation parameter structs. Path params are required strings; optional
// query filters are pointers (nil = omitted); date filters are *Date, rendered
// to the wire date format when non-nil (see setDate in client.go).
//
// The json tags carry the wire (snake_case) names so a JSON caller (e.g. an
// MCP server driving the OpenAPI spec) can decode straight into a params
// struct.

// Date is a date-only value. It (un)marshals as a bare "2006-01-02" JSON
// string, keeping external date types out of the public params. A nil *Date
// means the filter is omitted.
type Date struct {
	time.Time
}

const dateLayout = "2006-01-02"

// UnmarshalJSON parses a "2006-01-02" JSON string. A JSON null leaves the zero
// value in place (callers use *Date, so null is normally absence); any other
// non-date value (including an empty string) is a parse error, not a silent
// zero — so a required value-typed Date never decodes a bad value to
// 0001-01-01.
func (d *Date) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("parse date: %w", err)
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return fmt.Errorf("parse date %q (want YYYY-MM-DD): %w", s, err)
	}
	d.Time = t
	return nil
}

// MarshalJSON renders the date as a "2006-01-02" JSON string.
func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Format(dateLayout) + `"`), nil
}

// NewDate wraps t as a Date — the blessed way to build one in Go. The wire
// date is t's calendar day in t's own location, so pass a time already in the
// zone whose date you mean.
func NewDate(t time.Time) Date { return Date{Time: t} }

// mskZone is the location MOEX timestamps use. With time/tzdata embedded (see
// tzdata.go) LoadLocation resolves on any host; the FixedZone(+3) fallback is
// a last resort (e.g. a malformed ZONEINFO override) so ParseInLocation always
// has a non-nil location. It matches Moscow's current fixed +3/no-DST regime,
// so the instant stays correct if it fires. Loaded once at init.
var mskZone = func() *time.Location {
	if loc, err := time.LoadLocation("Europe/Moscow"); err == nil {
		return loc
	}
	return time.FixedZone("MSK", 3*60*60)
}()

const dateTimeLayout = "2006-01-02T15:04:05"

// DateTime is a MOEX timestamp. The wire form is a bare "2006-01-02T15:04:05"
// with no zone, but MOEX times are Moscow-local, so it (un)marshals through
// Europe/Moscow — the resulting instant is correct (.UTC(), .Unix(), and
// comparisons all hold), unlike a naive UTC parse. A nil *DateTime means the
// field is absent.
type DateTime struct {
	time.Time
}

// UnmarshalJSON parses a "2006-01-02T15:04:05" JSON string as Moscow-local
// time. A JSON null leaves the zero value in place (callers use *DateTime, so
// null is normally absence); any other non-timestamp value (including an empty
// string) is a parse error, not a silent zero.
func (d *DateTime) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("parse datetime: %w", err)
	}
	t, err := time.ParseInLocation(dateTimeLayout, s, mskZone)
	if err != nil {
		return fmt.Errorf(
			"parse datetime %q (want YYYY-MM-DDTHH:MM:SS): %w",
			s,
			err,
		)
	}
	d.Time = t
	return nil
}

// MarshalJSON renders the timestamp as a "2006-01-02T15:04:05" JSON string in
// the Moscow wall-clock, so a value from the wire round-trips to the exact
// string MOEX sent.
func (d DateTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.In(mskZone).Format(dateTimeLayout) + `"`), nil
}

// NewDateTime wraps t as a DateTime, normalizing to the Moscow zone so the
// instant is preserved and MarshalJSON emits the Moscow wall-clock — the same
// invariant UnmarshalJSON establishes. The blessed way to build one in Go.
func NewDateTime(t time.Time) DateTime { return DateTime{Time: t.In(mskZone)} }

type ListAssetsRequest struct {
	AssetType    *string `json:"asset_type,omitempty"`    // Тип базового актива.
	AssetSubtype *string `json:"asset_subtype,omitempty"` // Подтип базового актива для фьючерсов.
	Query        *string `json:"query,omitempty"`         // Фильтр названия БА.
}

type GetAssetRequest struct {
	AssetCode string  `json:"asset_code"`           // Торговый код БА.
	AssetType *string `json:"asset_type,omitempty"` // Тип базового актива.
}

type ListFuturesRequest struct {
	AssetCode      string `json:"asset_code"`                // Торговый код базового актива (фьючерса).
	ExpirationDate *Date  `json:"expiration_date,omitempty"` // Дата погашения.
}

type ListOptionsRequest struct {
	AssetCode      string   `json:"asset_code"`                // Торговый код БА.
	AssetType      *string  `json:"asset_type,omitempty"`      // Тип базового актива.
	ExpirationDate *Date    `json:"expiration_date,omitempty"` // Дата исполнения.
	OptionType     *string  `json:"option_type,omitempty"`     // Тип опциона (call/put).
	SeriesType     *string  `json:"series_type,omitempty"`     // Тип серии.
	Strike         *float64 `json:"strike,omitempty"`          // Страйк.
}

type GetOptionRequest struct {
	AssetCode         string   `json:"asset_code"`                    // Торговый код БА.
	Secid             string   `json:"secid"`                         // Торговый код опциона.
	AssetType         *string  `json:"asset_type,omitempty"`          // Тип базового актива.
	DaysUntilExpiring *int32   `json:"days_until_expiring,omitempty"` // Переоценка: дней до экспирации.
	UnderlyingPrice   *float64 `json:"underlying_price,omitempty"`    // Переоценка: цена БА.
	Volatility        *float64 `json:"volatility,omitempty"`          // Переоценка: волатильность.
}

type ListOptionSeriesRequest struct {
	AssetCode string  `json:"asset_code"`           // Торговый код БА.
	AssetType *string `json:"asset_type,omitempty"` // Тип базового актива.
}

type GetOptionSeriesRequest struct {
	AssetCode        string  `json:"asset_code"`           // Торговый код БА.
	OptionseriesCode string  `json:"optionseries_code"`    // Код серии опционов.
	AssetType        *string `json:"asset_type,omitempty"` // Тип базового актива.
}

type ListSeriesOptionsRequest struct {
	AssetCode        string  `json:"asset_code"`            // Торговый код БА.
	OptionseriesCode string  `json:"optionseries_code"`     // Код серии опционов.
	AssetType        *string `json:"asset_type,omitempty"`  // Тип базового актива.
	OptionType       *string `json:"option_type,omitempty"` // Тип опциона (call/put).
	// Strike is *int32 here (spec: integer) — deliberately unlike
	// ListOptionsRequest.Strike, which is *float64 (spec: number). Don't "fix"
	// the mismatch; the two endpoints declare different wire types.
	Strike *int32 `json:"strike,omitempty"` // Страйк.
}

type GetOptionBoardRequest struct {
	AssetCode        string  `json:"asset_code"`           // Торговый код БА.
	OptionseriesCode string  `json:"optionseries_code"`    // Код серии опционов.
	AssetType        *string `json:"asset_type,omitempty"` // Тип базового актива.
	Rows             *int32  `json:"rows,omitempty"`       // Число строк матрицы.
}

type GetVolatilityGraphRequest struct {
	AssetCode        string  `json:"asset_code"`           // Торговый код БА.
	OptionseriesCode string  `json:"optionseries_code"`    // Код серии опционов.
	AssetType        *string `json:"asset_type,omitempty"` // Тип базового актива.
}

type CalculatePortfolioGraphRequest struct {
	Indicator string `json:"indicator"` // Тип графика (path segment).
}
