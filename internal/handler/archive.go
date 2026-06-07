package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"
)

func (handler *Handler) ShowArchivedStories(ctx *echo.Context) error {
	boardID := ctx.Param("boardID")
	query := ctx.Request().URL.Query().Get("q")
	page, err := strconv.Atoi(ctx.Request().URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	result, err := handler.uc.ListArchivedStories(ctx.Request().Context(), boardID, query, page)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTML(ctx, http.StatusOK, "archived_stories.html", result)
}

func (handler *Handler) ArchiveDoneInColumn(ctx *echo.Context) error {
	columnID := ctx.Param("id")
	result, err := handler.uc.ArchiveDoneInColumn(ctx.Request().Context(), columnID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "archive_done_oob.html", result)
}
