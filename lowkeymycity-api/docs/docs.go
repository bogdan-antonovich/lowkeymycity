// Package docs carries the generated OpenAPI spec, embedded so the binary
// serves the exact spec it was built from. The json (and its yaml twin)
// are written by swag from the controller annotations — regenerate with:
//
//	swag init -g cmd/lowkeymycity/main.go -o docs --outputTypes json,yaml --parseDependency --parseInternal
//
// CI fails when the committed spec drifts from the annotations.
package docs

import _ "embed"

//go:embed swagger.json
var Swagger []byte
