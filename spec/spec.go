// Package spec embeds the dereferenced OpenAPI document from the codegen
// pipeline so downstream modules consume it without re-running Docker.
// SpecDerefJSON is the upstream MOEX option-calc spec fully dereferenced
// (components.schemas retained, every $ref inlined), suitable for a JSON
// Schema transformer needing self-contained per-operation schemas — e.g. the
// moexoptcalcmcp tool-schema builder.
//
// The ref-based, overlaid spec.json is oapi-codegen input only and is not
// embedded: nothing consumes it at runtime.
package spec

import _ "embed"

//go:embed spec-deref.json
var SpecDerefJSON []byte
