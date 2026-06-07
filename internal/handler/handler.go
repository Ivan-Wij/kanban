package handler

import (
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"kanban/internal/usecase"
)

type Handler struct {
	uc       *usecase.Kanban
	renderer *Renderer
}

func New(uc *usecase.Kanban, renderer *Renderer) *Handler {
	return &Handler{uc: uc, renderer: renderer}
}

func (handler *Handler) Register(echoServer *echo.Echo) {
	echoServer.Use(middleware.RequestLogger())
	echoServer.Use(middleware.Recover())

	echoServer.GET("/", handler.ShowBoardList)
	echoServer.GET("/boards/new", handler.ShowCreateBoardModal)
	echoServer.POST("/boards", handler.CreateBoard)
	echoServer.GET("/boards/:boardID", handler.ShowBoard)
	echoServer.PUT("/boards/:boardID", handler.UpdateBoard)
	echoServer.GET("/boards/:boardID/archived/stories", handler.ShowArchivedStories)
	echoServer.GET("/boards/:boardID/cards/new", handler.ShowCreateCardModal)
	echoServer.POST("/boards/:boardID/cards", handler.CreateCard)

	echoServer.GET("/columns/:id", handler.ShowColumn)
	echoServer.GET("/cards/:id/detail", handler.ShowCardDetail)
	echoServer.GET("/cards/:id", handler.ShowCard)
	echoServer.PUT("/cards/:id", handler.UpdateCard)
	echoServer.PUT("/cards/:id/status", handler.ChangeCardStatus)
	echoServer.DELETE("/cards/:id", handler.DeleteCard)
	echoServer.PUT("/cards/:id/move", handler.MoveCard)
	echoServer.POST("/columns/:id/archive-done", handler.ArchiveDoneInColumn)
}
