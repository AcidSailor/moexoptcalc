// Package moexoptcalc error types alias the shared restkit types, so callers
// match them with errors.As without importing restkit:
//
//	var re *moexoptcalc.ResponseError
//	if errors.As(err, &re) { _ = re.StatusCode } // e.g. 422
//
//	var re *moexoptcalc.RequestError
//	if errors.As(err, &re) { _ = re.Op } // OpMarshal/OpBuild/OpHook/OpSend/OpRead/OpUnmarshal
//
// No sentinel errors: the typed error IS the category.
package moexoptcalc

import "github.com/acidsailor/restkit"

// ResponseError is a non-2xx MOEX API response (status + raw body).
type ResponseError = restkit.ResponseError

// RequestError is a per-call failure; Op names the stage and Err wraps the cause.
type RequestError = restkit.RequestError

// ConfigError is invalid New construction input.
type ConfigError = restkit.ConfigError

// Op values name the lifecycle stage a RequestError failed at, so callers can
// switch on RequestError.Op without importing restkit:
//
//	var re *moexoptcalc.RequestError
//	if errors.As(err, &re) && re.Op == moexoptcalc.OpSend { /* round-trip failed */ }
const (
	OpMarshal   = restkit.OpMarshal   // encoding the request body to JSON
	OpBuild     = restkit.OpBuild     // constructing the *http.Request
	OpHook      = restkit.OpHook      // a request hook returned an error
	OpSend      = restkit.OpSend      // the HTTP round-trip
	OpRead      = restkit.OpRead      // reading the response body
	OpUnmarshal = restkit.OpUnmarshal // decoding the 2xx body into T
)
