package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"

	_ "github.com/jackc/pgx/v5/stdlib"

	"kanban/internal/config"
	"kanban/internal/handler"
	"kanban/internal/migrate"
	"kanban/internal/repository"
	"kanban/internal/usecase"
	"kanban/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := sqlx.Connect("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := migrate.Run(db, migrations.FS); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	repo := repository.NewPostgres(db)
	uc := usecase.NewKanban(repo)

	renderer, err := handler.NewRenderer()
	if err != nil {
		log.Fatalf("init renderer: %v", err)
	}

	echoServer := echo.New()
	echoServer.Renderer = renderer

	httpHandler := handler.New(uc, renderer)
	httpHandler.Register(echoServer)

	address := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("listening on http://localhost%s", address)
	if err := echoServer.Start(address); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
