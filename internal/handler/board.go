package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	"kanban/internal/domain"
)

type createBoardForm struct {
	Name            string `form:"name"`
	FromCreateModal string `form:"from_create_modal"`
}

type updateBoardForm struct {
	Name string `form:"name"`
}

func (handler *Handler) ShowBoardList(ctx *echo.Context) error {
	boards, err := handler.uc.ListBoards(ctx.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTML(ctx, http.StatusOK, "boards.html", domain.BoardListPage{Boards: boards})
}

func (handler *Handler) ShowFinishedBoards(ctx *echo.Context) error {
	query := ctx.Request().URL.Query().Get("q")
	sortField := ctx.Request().URL.Query().Get("sort")
	sortOrder := ctx.Request().URL.Query().Get("order")
	page, err := strconv.Atoi(ctx.Request().URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	result, err := handler.uc.ListFinishedBoards(ctx.Request().Context(), query, sortField, sortOrder, page)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTML(ctx, http.StatusOK, "finished_boards.html", result)
}

func (handler *Handler) RenderMarkdownPreview(ctx *echo.Context) error {
	text := ctx.FormValue("text")
	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "markdown_preview.html", text)
}

func (handler *Handler) ShowCreateBoardModal(ctx *echo.Context) error {
	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "create_board_modal.html", nil)
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
	if form.FromCreateModal == "true" {
		ctx.Response().Header().Set("HX-Redirect", "/boards/"+board.ID)
		return noContent(ctx)
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

func (handler *Handler) UpdateBoard(ctx *echo.Context) error {
	boardID := ctx.Param("boardID")
	var form updateBoardForm
	if err := ctx.Bind(&form); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if form.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	board, err := handler.uc.UpdateBoard(ctx.Request().Context(), boardID, form.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "board_header.html", board)
}

func (handler *Handler) DeleteBoard(ctx *echo.Context) error {
	boardID := ctx.Param("boardID")
	if err := handler.uc.DeleteBoard(ctx.Request().Context(), boardID); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	ctx.Response().Header().Set("HX-Redirect", "/")
	return noContent(ctx)
}

func (handler *Handler) FinishBoard(ctx *echo.Context) error {
	boardID := ctx.Param("boardID")
	board, err := handler.uc.FinishBoard(ctx.Request().Context(), boardID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "board_header.html", board)
}

func (handler *Handler) ShowColumn(ctx *echo.Context) error {
	columnID := ctx.Param("id")
	column, err := handler.uc.GetColumn(ctx.Request().Context(), columnID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "column.html", column)
}
