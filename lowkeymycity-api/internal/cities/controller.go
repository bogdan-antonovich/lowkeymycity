// Package cities serves the exported city list: the JSON file the city
// validator writes at startup, fetched by the frontend autocomplete. The
// controller serves the file verbatim and owns nothing else — the data,
// its ordering and its lifetime belong to the validator's Export.
package cities

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v5"
)

// Controller registers the cities HTTP routes on the versioned API group.
type Controller interface {
	GetCities(g *echo.Group)
}

// citiesController implements Controller around the path the validator
// exported the city list to, split into a directory filesystem and a name
// because echo's File resolves against the working directory and would
// 404 on the absolute CITIES_FILE path.
type citiesController struct {
	dir  fs.FS
	name string
}

// NewCitiesController builds the cities controller around path, the JSON
// file the validator's Export wrote. The returned Controller is inert
// until GetCities is called.
func NewCitiesController(path string) Controller {
	return &citiesController{
		dir:  os.DirFS(filepath.Dir(path)),
		name: filepath.Base(path),
	}
}

// GetCities registers GET /cities on g (mounted under the API prefix), serving the exported city
// list: every label the validator accepts, as a JSON array ordered by
// population descending — the order the frontend autocomplete ranks by.
// The set only changes with a reseed and a redeploy, so clients are told
// to cache it for a day. A missing file — Export hasn't run — surfaces as
// echo's 404.
//
// @Summary      US city list
// @Description  Every city label the validator accepts, ordered by population descending.
// @Tags         cities
// @Produce      json
// @Success      200  {array}   string
// @Failure      404  {object}  echo.HTTPError
// @Router       /cities [get]
func (cc *citiesController) GetCities(g *echo.Group) {
	g.GET("/cities", func(c *echo.Context) error {
		c.Response().Header().Set("Cache-Control", "public, max-age=86400")
		return c.FileFS(cc.name, cc.dir)
	})
}
