// Package apidocs serves the interactive API documentation: the embedded
// OpenAPI spec and the stock Swagger UI rendering it. The UI is the static
// swagger-ui distribution (embedded via swaggo/files); nothing here is
// project-specific except the index page pointing it at our spec.
//
// TODO: this package hand-rolls what swaggo/echo-swagger provides as a
// one-liner, because that adapter (v1.5.2 as of 2026-06) still targets
// echo v4 and we're on v5. When a v5-compatible release ships, replace
// this whole package with g.GET("/docs/*", echoSwagger.WrapHandler)
// and drop the swaggo/files/v2 direct dependency.
package apidocs

import (
	"net/http"
	"strings"

	"lowkeymycity/docs"
	"lowkeymycity/internal/server"

	"github.com/labstack/echo/v5"
	swaggerFiles "github.com/swaggo/files/v2"
)

// Controller registers the API documentation routes on the versioned API
// group.
type Controller interface {
	UI(g *echo.Group)
}

type docsController struct{}

// NewDocsController builds the docs controller. The returned Controller is
// inert until UI is called.
func NewDocsController() Controller {
	return &docsController{}
}

// docsBase is the full path the docs live under; the index page and the
// asset prefix-stripping both derive from it so the API prefix stays
// declared in exactly one place.
const docsBase = server.APIPrefix + "/docs"

// indexHTML boots Swagger UI against our spec. Asset and spec paths are
// absolute so GET <prefix>/docs works without a trailing slash.
var indexHTML = strings.ReplaceAll(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>LowkeyMyCity API docs</title>
  <link rel="stylesheet" href="{docs}/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="{docs}/swagger-ui-bundle.js"></script>
<script>
  window.ui = SwaggerUIBundle({
    url: '{docs}/swagger.json',
    dom_id: '#swagger-ui',
  });
</script>
</body>
</html>
`, "{docs}", docsBase)

// UI registers the documentation routes on g (mounted under the API
// prefix):
//
//	GET /docs              — the Swagger UI page
//	GET /docs/swagger.json — the embedded OpenAPI spec
//	GET /docs/*            — the swagger-ui static assets
//
// The dist's own index.html (and the trailing-slash root) answer with our
// index page instead, because the stock one points at the petstore demo.
func (dc *docsController) UI(g *echo.Group) {
	g.GET("/docs", func(c *echo.Context) error {
		return c.HTML(http.StatusOK, indexHTML)
	})

	g.GET("/docs/swagger.json", func(c *echo.Context) error {
		return c.Blob(http.StatusOK, "application/json", docs.Swagger)
	})

	assets := echo.WrapHandler(http.StripPrefix(docsBase+"/", http.FileServerFS(swaggerFiles.FS)))
	g.GET("/docs/*", func(c *echo.Context) error {
		switch c.Param("*") {
		case "", "index.html":
			return c.HTML(http.StatusOK, indexHTML)
		}
		return assets(c)
	})
}
