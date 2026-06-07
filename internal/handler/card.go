package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	"kanban/internal/domain"
)

type cardForm struct {
	Title       string `form:"title"`
	Description string `form:"description"`
	FromModal   string `form:"from_modal"`
}

type createCardForm struct {
	CardType        string `form:"card_type"`
	ParentCardID    string `form:"parent_card_id"`
	ModalCardID     string `form:"modal_card_id"`
	Title           string `form:"title"`
	Description     string `form:"description"`
	FromModal       string `form:"from_modal"`
	FromCreateModal string `form:"from_create_modal"`
}

func (handler *Handler) cardForBoard(ctx *echo.Context, cardID string) (domain.Card, error) {
	card, err := handler.uc.GetCard(ctx.Request().Context(), cardID)
	if err != nil {
		return domain.Card{}, err
	}

	column, err := handler.uc.GetColumn(ctx.Request().Context(), card.ColumnID)
	if err != nil {
		return domain.Card{}, err
	}

	for _, columnCard := range column.Cards {
		if columnCard.ID == cardID {
			return columnCard, nil
		}
	}

	return card, nil
}

func (handler *Handler) CreateCard(ctx *echo.Context) error {
	var form createCardForm
	if err := ctx.Bind(&form); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if form.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}

	cardType := domain.CardType(form.CardType)
	if !cardType.IsValid() {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid card type")
	}

	card, err := handler.uc.CreateCard(ctx.Request().Context(), cardType, form.ParentCardID, form.Title, form.Description)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if form.FromCreateModal == "true" {
		board, err := handler.uc.GetBoard(ctx.Request().Context())
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		column, err := handler.uc.GetColumn(ctx.Request().Context(), board.TodoColumnID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return handler.renderer.HTMLFragment(ctx, http.StatusOK, "create_card_board_oob.html", domain.CreateCardResult{
			Column:       column,
			TodoColumnID: board.TodoColumnID,
		})
	}

	if form.FromModal == "true" && form.ModalCardID != "" {
		detail, err := handler.uc.GetCardDetail(ctx.Request().Context(), form.ModalCardID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		column, err := handler.uc.GetColumn(ctx.Request().Context(), detail.TodoColumnID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return handler.renderer.HTMLFragment(ctx, http.StatusOK, "create_card_oob.html", domain.CreateCardResult{
			Column:       column,
			ModalDetail:  detail,
			FromModal:    true,
			TodoColumnID: detail.TodoColumnID,
		})
	}

	boardCard, err := handler.cardForBoard(ctx, card.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "card.html", boardCard)
}

func (handler *Handler) ShowCreateCardModal(ctx *echo.Context) error {
	board, err := handler.uc.GetBoard(ctx.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "create_card_modal.html", domain.CreateCardForm{
		Projects:     board.Projects,
		Stories:      board.Stories,
		TodoColumnID: board.TodoColumnID,
	})
}

func (handler *Handler) ShowCardDetail(ctx *echo.Context) error {
	cardID := ctx.Param("id")
	detail, err := handler.uc.GetCardDetail(ctx.Request().Context(), cardID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "card_detail.html", detail)
}

func (handler *Handler) ShowCard(ctx *echo.Context) error {
	cardID := ctx.Param("id")
	boardCard, err := handler.cardForBoard(ctx, cardID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "card.html", boardCard)
}

func (handler *Handler) UpdateCard(ctx *echo.Context) error {
	cardID := ctx.Param("id")
	var form cardForm
	if err := ctx.Bind(&form); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if form.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}

	_, err := handler.uc.UpdateCard(ctx.Request().Context(), cardID, form.Title, form.Description)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if form.FromModal == "true" {
		detail, err := handler.uc.GetCardDetail(ctx.Request().Context(), cardID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		column, err := handler.uc.GetColumn(ctx.Request().Context(), detail.ColumnID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return handler.renderer.HTMLFragment(ctx, http.StatusOK, "update_card_oob.html", domain.CreateCardResult{
			Column:       column,
			ModalDetail:  detail,
			TodoColumnID: detail.TodoColumnID,
		})
	}

	boardCard, err := handler.cardForBoard(ctx, cardID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "card.html", boardCard)
}

func (handler *Handler) DeleteCard(ctx *echo.Context) error {
	cardID := ctx.Param("id")
	fromModal := ctx.Request().FormValue("from_modal") == "true"

	result, err := handler.uc.DeleteCard(ctx.Request().Context(), cardID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if fromModal {
		return handler.renderer.HTMLFragment(ctx, http.StatusOK, "delete_card_oob.html", result)
	}

	return noContent(ctx)
}

func (handler *Handler) ChangeCardStatus(ctx *echo.Context) error {
	cardID := ctx.Param("id")
	columnID := ctx.Request().FormValue("column_id")
	if columnID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "column_id is required")
	}

	result, err := handler.uc.ChangeCardStatus(ctx.Request().Context(), cardID, columnID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "status_change_oob.html", result)
}

func (handler *Handler) MoveCard(ctx *echo.Context) error {
	cardID := ctx.Param("id")
	columnID := ctx.Request().FormValue("column_id")
	position, err := strconv.Atoi(ctx.Request().FormValue("position"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid position")
	}
	if columnID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "column_id is required")
	}

	if err := handler.uc.MoveCard(ctx.Request().Context(), cardID, columnID, position); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	column, err := handler.uc.GetColumn(ctx.Request().Context(), columnID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return handler.renderer.HTMLFragment(ctx, http.StatusOK, "column.html", column)
}
