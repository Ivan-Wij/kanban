package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func (handler *Handler) ShowBoard(ctx *echo.Context) error {
	board, err := handler.uc.GetBoard(ctx.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTML(ctx, http.StatusOK, "board.html", board)
}

func (handler *Handler) ShowColumn(ctx *echo.Context) error {
	columnID := ctx.Param("id")
	column, err := handler.uc.GetColumn(ctx.Request().Context(), columnID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "column.html", column)
}
