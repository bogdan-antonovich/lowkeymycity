package results

import (
	"errors"
	"net/http"

	"lowkeymycity/pkg/logging"
	"lowkeymycity/pkg/pdf"
	_ "lowkeymycity/pkg/types"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

// ResultsController registers the stored-result HTTP routes on an Echo
// instance.
type ResultsController interface {
	GetResult(g *echo.Group)
	GetPDF(g *echo.Group)
}

// resultsController implements ResultsController; its handlers load the
// stored result through qs and only differ in how they answer — JSON or
// rendered PDF.
type resultsController struct {
	qs ResultsService
}

// NewResultsController builds the results controller around qs, which must
// be a non-nil ResultsService — nil isn't rejected here but panics on the
// first request. The returned controller is inert until its
// route-registering methods are called.
func NewResultsController(qs ResultsService) ResultsController {
	return &resultsController{qs: qs}
}

// GetResult registers GET /results/:id on g (mounted under the API prefix), the JSON source for the
// shareable result page. The id path parameter is the permanent id issued
// when a quiz was submitted; the stored verdict comes back as a 200 with
// the full QuizResult. An id that was never issued (or whose result was
// cleaned up) is a 404; any database failure surfaces as a 500.
//
// @Summary      Stored quiz result
// @Description  The stored verdict for one result id, as shown on the result page.
// @Tags         results
// @Produce      json
// @Param        id  path  string  true  "result id"
// @Success      200  {object}  types.QuizResult
// @Failure      404  {object}  echo.HTTPError
// @Failure      500  {object}  echo.HTTPError
// @Router       /results/{id} [get]
func (rc *resultsController) GetResult(g *echo.Group) {
	g.GET("/results/:id", func(c *echo.Context) error {
		id := c.Param("id")
		logging.From(c.Request().Context()).Debug("GET /results/:id", zap.String("id", id))

		result, err := rc.qs.GetResult(c.Request().Context(), id)
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "no result with that id")
		}
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, result)
	})
}

// GetPDF registers GET /results/:id/pdf on g (mounted under the API prefix), the "keep the receipts"
// download. It loads the stored result for the id path parameter, renders
// it as the one-page A4 PDF matching the site's print layout, and answers
// 200 with the PDF bytes under an attachment Content-Disposition, so
// browsers save it as lowkeymycity.pdf. An unknown id is a 404; a database
// failure or a rendering failure surfaces as a 500.
//
// @Summary      Result as PDF
// @Description  The stored verdict rendered as a one-page A4 PDF ("keep the receipts").
// @Tags         results
// @Produce      application/pdf
// @Param        id  path  string  true  "result id"
// @Success      200  {file}  file
// @Failure      404  {object}  echo.HTTPError
// @Failure      500  {object}  echo.HTTPError
// @Router       /results/{id}/pdf [get]
func (rc *resultsController) GetPDF(g *echo.Group) {
	g.GET("/results/:id/pdf", func(c *echo.Context) error {
		id := c.Param("id")
		log := logging.From(c.Request().Context())
		log.Debug("GET /results/:id/pdf", zap.String("id", id))

		result, err := rc.qs.GetResult(c.Request().Context(), id)
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "no result with that id")
		}
		if err != nil {
			return err
		}

		doc, err := pdf.Render(result)
		if err != nil {
			log.Error("rendering result PDF", zap.Error(err), zap.String("id", id))
			return err
		}

		c.Response().Header().Set("Content-Disposition", `attachment; filename="lowkeymycity.pdf"`)
		return c.Blob(http.StatusOK, "application/pdf", doc)
	})
}
