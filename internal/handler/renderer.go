package handler

import (
	"html/template"
	"io"
	"net/http"

	"github.com/labstack/echo/v5"

	"kanban/web/templates"
)

type Renderer struct {
	templates *template.Template
}

func NewRenderer() (*Renderer, error) {
	parsed, err := template.ParseFS(templates.FS, "*.html", "partials/*.html")
	if err != nil {
		return nil, err
	}
	return &Renderer{templates: parsed}, nil
}

func (renderer *Renderer) Render(_ *echo.Context, writer io.Writer, name string, data interface{}) error {
	return renderer.templates.ExecuteTemplate(writer, name, data)
}

func (renderer *Renderer) HTML(ctx *echo.Context, status int, name string, data interface{}) error {
	ctx.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	ctx.Response().WriteHeader(status)
	return renderer.templates.ExecuteTemplate(ctx.Response(), name, data)
}

func (renderer *Renderer) HTMLFragment(ctx *echo.Context, status int, name string, data interface{}) error {
	return renderer.HTML(ctx, status, name, data)
}

func noContent(ctx *echo.Context) error {
	return ctx.NoContent(http.StatusOK)
}
