package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"kanban/internal/domain"
)

type Postgres struct {
	db *sqlx.DB
}

func NewPostgres(db *sqlx.DB) *Postgres {
	return &Postgres{db: db}
}

const (
	defaultBoardID = "00000000-0000-0000-0000-000000000001"
	todoColumnID   = "00000000-0000-0000-0000-000000000011"
)

func (repo *Postgres) GetBoardWithColumnsAndCards(ctx context.Context) (domain.Board, error) {
	var board domain.Board
	if err := repo.db.GetContext(ctx, &board, `
		SELECT id, name, created_at FROM boards WHERE id = $1
	`, defaultBoardID); err != nil {
		return domain.Board{}, fmt.Errorf("get board: %w", err)
	}

	var columns []domain.Column
	if err := repo.db.SelectContext(ctx, &columns, `
		SELECT id, board_id, name, position, created_at
		FROM columns
		WHERE board_id = $1
		ORDER BY position ASC
	`, defaultBoardID); err != nil {
		return domain.Board{}, fmt.Errorf("get columns: %w", err)
	}

	var cards []domain.Card
	if err := repo.db.SelectContext(ctx, &cards, `
		SELECT c.id, c.column_id, c.card_type, c.parent_card_id, c.title, c.description, c.position, c.created_at
		FROM cards c
		INNER JOIN columns col ON col.id = c.column_id
		WHERE col.board_id = $1
		ORDER BY c.position ASC
	`, defaultBoardID); err != nil {
		return domain.Board{}, fmt.Errorf("get cards: %w", err)
	}

	parentTitles := make(map[string]string)
	for _, card := range cards {
		parentTitles[card.ID] = card.Title
	}
	for index := range cards {
		if cards[index].ParentCardID != "" {
			cards[index].ParentTitle = parentTitles[cards[index].ParentCardID]
		}
	}

	cardsByColumn := make(map[string][]domain.Card)
	var projects []domain.Card
	var stories []domain.Card
	for _, card := range cards {
		cardsByColumn[card.ColumnID] = append(cardsByColumn[card.ColumnID], card)
		switch card.CardType {
		case domain.CardTypeProject:
			projects = append(projects, card)
		case domain.CardTypeStory:
			stories = append(stories, card)
		}
	}

	for index := range columns {
		columnCards := cardsByColumn[columns[index].ID]
		if columnCards == nil {
			columnCards = []domain.Card{}
		}
		for cardIndex := range columnCards {
			columnCards[cardIndex].ColumnName = columns[index].Name
		}
		columns[index].Cards = domain.PrepareColumnCards(columnCards)
	}

	board.Columns = columns
	board.TodoColumnID = todoColumnID
	board.Projects = projects
	if board.Projects == nil {
		board.Projects = []domain.Card{}
	}
	board.Stories = stories
	if board.Stories == nil {
		board.Stories = []domain.Card{}
	}
	return board, nil
}

func (repo *Postgres) GetColumnWithCards(ctx context.Context, columnID string) (domain.Column, error) {
	var column domain.Column
	if err := repo.db.GetContext(ctx, &column, `
		SELECT id, board_id, name, position, created_at
		FROM columns WHERE id = $1
	`, columnID); err != nil {
		return domain.Column{}, fmt.Errorf("get column: %w", err)
	}

	var cards []domain.Card
	if err := repo.db.SelectContext(ctx, &cards, `
		SELECT id, column_id, card_type, parent_card_id, title, description, position, created_at
		FROM cards
		WHERE column_id = $1
		ORDER BY position ASC
	`, columnID); err != nil {
		return domain.Column{}, fmt.Errorf("get cards: %w", err)
	}
	if cards == nil {
		cards = []domain.Card{}
	}
	for index := range cards {
		cards[index].ColumnName = column.Name
		if cards[index].ParentCardID != "" {
			_ = repo.db.GetContext(ctx, &cards[index].ParentTitle, `SELECT title FROM cards WHERE id = $1`, cards[index].ParentCardID)
		}
	}
	column.Cards = domain.PrepareColumnCards(cards)
	return column, nil
}

func (repo *Postgres) CreateCard(ctx context.Context, cardType domain.CardType, parentCardID, title, description string) (domain.Card, error) {
	if err := repo.validateCardHierarchy(ctx, cardType, parentCardID); err != nil {
		return domain.Card{}, err
	}

	var maxPosition int
	if err := repo.db.GetContext(ctx, &maxPosition, `
		SELECT COALESCE(MAX(position), -1) FROM cards WHERE column_id = $1
	`, todoColumnID); err != nil {
		return domain.Card{}, fmt.Errorf("get max position: %w", err)
	}

	card := domain.Card{
		ID:           uuid.New().String(),
		ColumnID:     todoColumnID,
		CardType:     cardType,
		ParentCardID: parentCardID,
		Title:        title,
		Description:  description,
		Position:     maxPosition + 1,
	}

	_, err := repo.db.NamedExecContext(ctx, `
		INSERT INTO cards (id, column_id, card_type, parent_card_id, title, description, position)
		VALUES (:id, :column_id, :card_type, :parent_card_id, :title, :description, :position)
	`, card)
	if err != nil {
		return domain.Card{}, fmt.Errorf("insert card: %w", err)
	}

	if err := repo.db.GetContext(ctx, &card.CreatedAt, `SELECT created_at FROM cards WHERE id = $1`, card.ID); err != nil {
		return domain.Card{}, fmt.Errorf("get created_at: %w", err)
	}

	if card.ParentCardID != "" {
		if err := repo.db.GetContext(ctx, &card.ParentTitle, `SELECT title FROM cards WHERE id = $1`, card.ParentCardID); err != nil {
			return domain.Card{}, fmt.Errorf("get parent title: %w", err)
		}
	}

	if err := repo.db.GetContext(ctx, &card.ColumnName, `SELECT name FROM columns WHERE id = $1`, card.ColumnID); err != nil {
		return domain.Card{}, fmt.Errorf("get column name: %w", err)
	}

	return card, nil
}

func (repo *Postgres) validateCardHierarchy(ctx context.Context, cardType domain.CardType, parentCardID string) error {
	if !cardType.IsValid() {
		return fmt.Errorf("invalid card type")
	}

	switch cardType {
	case domain.CardTypeProject:
		if parentCardID != "" {
			return fmt.Errorf("project cannot have a parent")
		}
	case domain.CardTypeStory:
		if parentCardID == "" {
			return fmt.Errorf("story requires a parent project")
		}
		parent, err := repo.GetCard(ctx, parentCardID)
		if err != nil {
			return fmt.Errorf("parent project not found")
		}
		if parent.CardType != domain.CardTypeProject {
			return fmt.Errorf("story parent must be a project")
		}
	case domain.CardTypeTask:
		if parentCardID == "" {
			return fmt.Errorf("task requires a parent story")
		}
		parent, err := repo.GetCard(ctx, parentCardID)
		if err != nil {
			return fmt.Errorf("parent story not found")
		}
		if parent.CardType != domain.CardTypeStory {
			return fmt.Errorf("task parent must be a story")
		}
	}
	return nil
}

func (repo *Postgres) GetBoardColumns(ctx context.Context) ([]domain.Column, error) {
	var columns []domain.Column
	if err := repo.db.SelectContext(ctx, &columns, `
		SELECT id, board_id, name, position, created_at
		FROM columns
		WHERE board_id = $1
		ORDER BY position ASC
	`, defaultBoardID); err != nil {
		return nil, fmt.Errorf("get board columns: %w", err)
	}
	if columns == nil {
		columns = []domain.Column{}
	}
	return columns, nil
}

func (repo *Postgres) GetCardDetail(ctx context.Context, cardID string) (domain.CardDetail, error) {
	card, err := repo.GetCard(ctx, cardID)
	if err != nil {
		return domain.CardDetail{}, err
	}

	var columnName string
	if err := repo.db.GetContext(ctx, &columnName, `
		SELECT name FROM columns WHERE id = $1
	`, card.ColumnID); err != nil {
		return domain.CardDetail{}, fmt.Errorf("get column name: %w", err)
	}

	columns, err := repo.GetBoardColumns(ctx)
	if err != nil {
		return domain.CardDetail{}, err
	}

	var children []domain.Card
	if err := repo.db.SelectContext(ctx, &children, `
		SELECT c.id, c.column_id, c.card_type, c.parent_card_id, c.title, c.description, c.position, c.created_at, col.name AS column_name
		FROM cards c
		INNER JOIN columns col ON col.id = c.column_id
		WHERE c.parent_card_id = $1
		ORDER BY c.created_at ASC
	`, cardID); err != nil {
		return domain.CardDetail{}, fmt.Errorf("get child cards: %w", err)
	}
	if children == nil {
		children = []domain.Card{}
	}

	return domain.CardDetail{
		Card:         card,
		ColumnName:   columnName,
		Columns:      columns,
		Children:     children,
		TodoColumnID: todoColumnID,
	}, nil
}

func (repo *Postgres) GetCard(ctx context.Context, cardID string) (domain.Card, error) {
	var card domain.Card
	if err := repo.db.GetContext(ctx, &card, `
		SELECT id, column_id, card_type, parent_card_id, title, description, position, created_at
		FROM cards WHERE id = $1
	`, cardID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Card{}, fmt.Errorf("card not found")
		}
		return domain.Card{}, fmt.Errorf("get card: %w", err)
	}
	if card.ParentCardID != "" {
		_ = repo.db.GetContext(ctx, &card.ParentTitle, `SELECT title FROM cards WHERE id = $1`, card.ParentCardID)
	}
	_ = repo.db.GetContext(ctx, &card.ColumnName, `SELECT name FROM columns WHERE id = $1`, card.ColumnID)
	return card, nil
}

func (repo *Postgres) UpdateCard(ctx context.Context, cardID, title, description string) (domain.Card, error) {
	_, err := repo.db.ExecContext(ctx, `
		UPDATE cards SET title = $1, description = $2 WHERE id = $3
	`, title, description, cardID)
	if err != nil {
		return domain.Card{}, fmt.Errorf("update card: %w", err)
	}
	return repo.GetCard(ctx, cardID)
}

func (repo *Postgres) DeleteCard(ctx context.Context, cardID string) (domain.DeleteCardResult, error) {
	if _, err := repo.GetCard(ctx, cardID); err != nil {
		return domain.DeleteCardResult{}, err
	}

	descendantIDs, err := repo.getDescendantCardIDs(ctx, cardID)
	if err != nil {
		return domain.DeleteCardResult{}, err
	}

	allIDs := append(descendantIDs, cardID)

	deleteQuery, deleteArgs, err := sqlx.In(`DELETE FROM cards WHERE id IN (?)`, allIDs)
	if err != nil {
		return domain.DeleteCardResult{}, fmt.Errorf("build delete query: %w", err)
	}
	deleteQuery = repo.db.Rebind(deleteQuery)

	columnsQuery, columnsArgs, err := sqlx.In(`SELECT DISTINCT column_id FROM cards WHERE id IN (?)`, allIDs)
	if err != nil {
		return domain.DeleteCardResult{}, fmt.Errorf("build columns query: %w", err)
	}
	columnsQuery = repo.db.Rebind(columnsQuery)

	var affectedColumnIDs []string
	if err := repo.db.SelectContext(ctx, &affectedColumnIDs, columnsQuery, columnsArgs...); err != nil {
		return domain.DeleteCardResult{}, fmt.Errorf("get affected columns: %w", err)
	}

	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return domain.DeleteCardResult{}, fmt.Errorf("begin delete transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
		return domain.DeleteCardResult{}, fmt.Errorf("delete cards: %w", err)
	}

	for _, columnID := range affectedColumnIDs {
		if _, err := tx.ExecContext(ctx, `
			WITH ordered AS (
				SELECT id, ROW_NUMBER() OVER (ORDER BY position ASC) - 1 AS new_position
				FROM cards
				WHERE column_id = $1
			)
			UPDATE cards
			SET position = ordered.new_position
			FROM ordered
			WHERE cards.id = ordered.id
		`, columnID); err != nil {
			return domain.DeleteCardResult{}, fmt.Errorf("reorder column %s: %w", columnID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.DeleteCardResult{}, fmt.Errorf("commit delete: %w", err)
	}
	return domain.DeleteCardResult{DeletedIDs: allIDs}, nil
}

func (repo *Postgres) getDescendantCardIDs(ctx context.Context, cardID string) ([]string, error) {
	var childIDs []string
	if err := repo.db.SelectContext(ctx, &childIDs, `
		SELECT id FROM cards WHERE parent_card_id = $1
	`, cardID); err != nil {
		return nil, fmt.Errorf("get child cards: %w", err)
	}

	var descendantIDs []string
	for _, childID := range childIDs {
		descendantIDs = append(descendantIDs, childID)
		grandchildIDs, err := repo.getDescendantCardIDs(ctx, childID)
		if err != nil {
			return nil, err
		}
		descendantIDs = append(descendantIDs, grandchildIDs...)
	}
	return descendantIDs, nil
}

func (repo *Postgres) MoveCard(ctx context.Context, cardID, toColumnID string, position int) error {
	card, err := repo.GetCard(ctx, cardID)
	if err != nil {
		return err
	}

	fromColumnID := card.ColumnID
	if position < 0 {
		return fmt.Errorf("position must be non-negative")
	}

	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin move transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if fromColumnID == toColumnID {
		if position > card.Position {
			position--
		}
		if position == card.Position {
			return tx.Commit()
		}

		if card.Position < position {
			if _, err := tx.ExecContext(ctx, `
				UPDATE cards SET position = position - 1
				WHERE column_id = $1 AND position > $2 AND position <= $3 AND id != $4
			`, fromColumnID, card.Position, position, cardID); err != nil {
				return fmt.Errorf("shift down same column: %w", err)
			}
		} else {
			if _, err := tx.ExecContext(ctx, `
				UPDATE cards SET position = position + 1
				WHERE column_id = $1 AND position >= $2 AND position < $3 AND id != $4
			`, fromColumnID, position, card.Position, cardID); err != nil {
				return fmt.Errorf("shift up same column: %w", err)
			}
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE cards SET position = $1 WHERE id = $2
		`, position, cardID); err != nil {
			return fmt.Errorf("update position same column: %w", err)
		}

		return tx.Commit()
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE cards SET position = position - 1
		WHERE column_id = $1 AND position > $2
	`, fromColumnID, card.Position); err != nil {
		return fmt.Errorf("close gap source column: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE cards SET position = position + 1
		WHERE column_id = $1 AND position >= $2
	`, toColumnID, position); err != nil {
		return fmt.Errorf("make space target column: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE cards SET column_id = $1, position = $2 WHERE id = $3
	`, toColumnID, position, cardID); err != nil {
		return fmt.Errorf("move card: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit move: %w", err)
	}
	return nil
}
