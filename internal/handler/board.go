package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"kanban/internal/domain"
)

type createBoardForm struct {
	Name string `form:"name"`
}

func (handler *Handler) ShowBoardList(ctx *echo.Context) error {
	boards, err := handler.uc.ListBoards(ctx.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTML(ctx, http.StatusOK, "boards.html", domain.BoardListPage{Boards: boards})
}

func (handler *Handler) CreateBoard(ctx *echo.Context) error {
	var form createBoardForm
	if err := ctx.Bind(&form); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if form.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	board, err := handler.uc.CreateBoard(ctx.Request().Context(), form.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return ctx.Redirect(http.StatusSeeOther, "/boards/"+board.ID)
}

func (handler *Handler) ShowBoard(ctx *echo.Context) error {
	boardID := ctx.Param("boardID")
	board, err := handler.uc.GetBoard(ctx.Request().Context(), boardID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
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
