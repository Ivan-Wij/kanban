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

	echoServer.GET("/", handler.ShowBoard)
	echoServer.GET("/columns/:id", handler.ShowColumn)
	echoServer.GET("/cards/new", handler.ShowCreateCardModal)
	echoServer.POST("/cards", handler.CreateCard)
	echoServer.GET("/cards/:id/detail", handler.ShowCardDetail)
	echoServer.GET("/cards/:id", handler.ShowCard)
	echoServer.PUT("/cards/:id", handler.UpdateCard)
	echoServer.PUT("/cards/:id/status", handler.ChangeCardStatus)
	echoServer.DELETE("/cards/:id", handler.DeleteCard)
	echoServer.PUT("/cards/:id/move", handler.MoveCard)
	echoServer.GET("/archived/stories", handler.ShowArchivedStories)
	echoServer.POST("/columns/:id/archive-done", handler.ArchiveDoneInColumn)
}
