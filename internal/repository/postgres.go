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

const archivedPageSize = 20

var defaultColumnNames = []string{"To Do", "In Progress", "Done"}

func (repo *Postgres) ensureBoardAccessible(ctx context.Context, boardID string) error {
	var deleted bool
	if err := repo.db.GetContext(ctx, &deleted, `
		SELECT deleted FROM boards WHERE id = $1
	`, boardID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("board not found")
		}
		return fmt.Errorf("check board: %w", err)
	}
	if deleted {
		return fmt.Errorf("board not found")
	}
	return nil
}

func (repo *Postgres) ListBoards(ctx context.Context) ([]domain.Board, error) {
	var boards []domain.Board
	if err := repo.db.SelectContext(ctx, &boards, `
		SELECT id, name, created_at, finished, deleted
		FROM boards
		WHERE deleted = false AND finished = false
		ORDER BY created_at DESC
	`); err != nil {
		return nil, fmt.Errorf("list boards: %w", err)
	}
	if boards == nil {
		boards = []domain.Board{}
	}
	return boards, nil
}

func (repo *Postgres) CreateBoard(ctx context.Context, name string) (domain.Board, error) {
	boardID := uuid.New().String()

	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return domain.Board{}, fmt.Errorf("begin create board transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO boards (id, name) VALUES ($1, $2)
	`, boardID, name); err != nil {
		return domain.Board{}, fmt.Errorf("insert board: %w", err)
	}

	for position, columnName := range defaultColumnNames {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO columns (id, board_id, name, position) VALUES ($1, $2, $3, $4)
		`, uuid.New().String(), boardID, columnName, position); err != nil {
			return domain.Board{}, fmt.Errorf("insert column: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.Board{}, fmt.Errorf("commit create board: %w", err)
	}
	return repo.GetBoardWithColumnsAndCards(ctx, boardID)
}

func (repo *Postgres) UpdateBoard(ctx context.Context, boardID, name string) (domain.Board, error) {
	if err := repo.ensureBoardAccessible(ctx, boardID); err != nil {
		return domain.Board{}, err
	}

	result, err := repo.db.ExecContext(ctx, `
		UPDATE boards SET name = $1 WHERE id = $2 AND deleted = false
	`, name, boardID)
	if err != nil {
		return domain.Board{}, fmt.Errorf("update board: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.Board{}, fmt.Errorf("update board rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.Board{}, fmt.Errorf("board not found")
	}

	var board domain.Board
	if err := repo.db.GetContext(ctx, &board, `
		SELECT id, name, created_at, finished, deleted FROM boards WHERE id = $1
	`, boardID); err != nil {
		return domain.Board{}, fmt.Errorf("get updated board: %w", err)
	}
	return board, nil
}

func (repo *Postgres) DeleteBoard(ctx context.Context, boardID string) error {
	if err := repo.ensureBoardAccessible(ctx, boardID); err != nil {
		return err
	}

	result, err := repo.db.ExecContext(ctx, `
		UPDATE boards SET deleted = true WHERE id = $1 AND deleted = false
	`, boardID)
	if err != nil {
		return fmt.Errorf("delete board: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete board rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("board not found")
	}
	return nil
}

func (repo *Postgres) FinishBoard(ctx context.Context, boardID string) (domain.Board, error) {
	if err := repo.ensureBoardAccessible(ctx, boardID); err != nil {
		return domain.Board{}, err
	}

	result, err := repo.db.ExecContext(ctx, `
		UPDATE boards SET finished = true WHERE id = $1 AND deleted = false
	`, boardID)
	if err != nil {
		return domain.Board{}, fmt.Errorf("finish board: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.Board{}, fmt.Errorf("finish board rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.Board{}, fmt.Errorf("board not found")
	}

	var board domain.Board
	if err := repo.db.GetContext(ctx, &board, `
		SELECT id, name, created_at, finished, deleted FROM boards WHERE id = $1
	`, boardID); err != nil {
		return domain.Board{}, fmt.Errorf("get finished board: %w", err)
	}
	return board, nil
}

func (repo *Postgres) getTodoColumnID(ctx context.Context, boardID string) (string, error) {
	var columnID string
	if err := repo.db.GetContext(ctx, &columnID, `
		SELECT id FROM columns WHERE board_id = $1 AND position = 0
	`, boardID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("todo column not found")
		}
		return "", fmt.Errorf("get todo column: %w", err)
	}
	return columnID, nil
}

func (repo *Postgres) getCardBoardID(ctx context.Context, cardID string) (string, error) {
	var boardID string
	if err := repo.db.GetContext(ctx, &boardID, `
		SELECT col.board_id
		FROM cards c
		INNER JOIN columns col ON col.id = c.column_id
		INNER JOIN boards board ON board.id = col.board_id
		WHERE c.id = $1 AND board.deleted = false
	`, cardID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("card not found")
		}
		return "", fmt.Errorf("get card board: %w", err)
	}
	return boardID, nil
}

func (repo *Postgres) getColumnBoardID(ctx context.Context, columnID string) (string, error) {
	var boardID string
	if err := repo.db.GetContext(ctx, &boardID, `
		SELECT col.board_id
		FROM columns col
		INNER JOIN boards board ON board.id = col.board_id
		WHERE col.id = $1 AND board.deleted = false
	`, columnID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("column not found")
		}
		return "", fmt.Errorf("get column board: %w", err)
	}
	return boardID, nil
}

func (repo *Postgres) GetBoardWithColumnsAndCards(ctx context.Context, boardID string) (domain.Board, error) {
	var board domain.Board
	if err := repo.db.GetContext(ctx, &board, `
		SELECT id, name, created_at, finished, deleted
		FROM boards
		WHERE id = $1 AND deleted = false
	`, boardID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Board{}, fmt.Errorf("board not found")
		}
		return domain.Board{}, fmt.Errorf("get board: %w", err)
	}

	var columns []domain.Column
	if err := repo.db.SelectContext(ctx, &columns, `
		SELECT id, board_id, name, position, created_at
		FROM columns
		WHERE board_id = $1
		ORDER BY position ASC
	`, boardID); err != nil {
		return domain.Board{}, fmt.Errorf("get columns: %w", err)
	}

	var cards []domain.Card
	if err := repo.db.SelectContext(ctx, &cards, `
		SELECT c.id, c.column_id, c.card_type, c.parent_card_id, c.title, c.description, c.position, c.created_at, c.archived
		FROM cards c
		INNER JOIN columns col ON col.id = c.column_id
		WHERE col.board_id = $1 AND c.archived = false
		ORDER BY c.position ASC
	`, boardID); err != nil {
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

	cardIndex := domain.BuildCardIndex(cards)
	domain.EnrichProjectCardIDs(cards, cardIndex)

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

	todoColumnID, err := repo.getTodoColumnID(ctx, boardID)
	if err != nil {
		return domain.Board{}, err
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
	if _, err := repo.getColumnBoardID(ctx, columnID); err != nil {
		return domain.Column{}, err
	}

	var column domain.Column
	if err := repo.db.GetContext(ctx, &column, `
		SELECT id, board_id, name, position, created_at
		FROM columns WHERE id = $1
	`, columnID); err != nil {
		return domain.Column{}, fmt.Errorf("get column: %w", err)
	}

	var cards []domain.Card
	if err := repo.db.SelectContext(ctx, &cards, `
		SELECT id, column_id, card_type, parent_card_id, title, description, position, created_at, archived
		FROM cards
		WHERE column_id = $1 AND archived = false
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

	cardIndex, err := repo.getCardHierarchyIndex(ctx, column.BoardID)
	if err != nil {
		return domain.Column{}, err
	}
	domain.EnrichProjectCardIDs(cards, cardIndex)

	column.Cards = domain.PrepareColumnCards(cards)
	return column, nil
}

func (repo *Postgres) getCardHierarchyIndex(ctx context.Context, boardID string) (map[string]domain.Card, error) {
	var cards []domain.Card
	if err := repo.db.SelectContext(ctx, &cards, `
		SELECT c.id, c.card_type, c.parent_card_id
		FROM cards c
		INNER JOIN columns col ON col.id = c.column_id
		WHERE col.board_id = $1
	`, boardID); err != nil {
		return nil, fmt.Errorf("get card hierarchy: %w", err)
	}
	return domain.BuildCardIndex(cards), nil
}

func (repo *Postgres) CreateCard(ctx context.Context, boardID string, cardType domain.CardType, parentCardID, title, description string) (domain.Card, error) {
	if err := repo.ensureBoardAccessible(ctx, boardID); err != nil {
		return domain.Card{}, err
	}
	if err := repo.validateCardHierarchy(ctx, boardID, cardType, parentCardID); err != nil {
		return domain.Card{}, err
	}

	todoColumnID, err := repo.getTodoColumnID(ctx, boardID)
	if err != nil {
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

	_, err = repo.db.NamedExecContext(ctx, `
		INSERT INTO cards (id, column_id, card_type, parent_card_id, title, description, position)
		VALUES (:id, :column_id, :card_type, :parent_card_id, :title, :description, :position)
	`, card)
	if err != nil {
		return domain.Card{}, fmt.Errorf("insert card: %w", err)
	}

	if err = repo.db.GetContext(ctx, &card.CreatedAt, `SELECT created_at FROM cards WHERE id = $1`, card.ID); err != nil {
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

func (repo *Postgres) validateCardHierarchy(ctx context.Context, boardID string, cardType domain.CardType, parentCardID string) error {
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
		if err := repo.ensureCardOnBoard(ctx, parentCardID, boardID); err != nil {
			return err
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
		if err := repo.ensureCardOnBoard(ctx, parentCardID, boardID); err != nil {
			return err
		}
	}
	return nil
}

func (repo *Postgres) ensureCardOnBoard(ctx context.Context, cardID, boardID string) error {
	cardBoardID, err := repo.getCardBoardID(ctx, cardID)
	if err != nil {
		return err
	}
	if cardBoardID != boardID {
		return fmt.Errorf("parent card belongs to a different board")
	}
	return nil
}

func (repo *Postgres) GetBoardColumns(ctx context.Context, boardID string) ([]domain.Column, error) {
	var columns []domain.Column
	if err := repo.db.SelectContext(ctx, &columns, `
		SELECT id, board_id, name, position, created_at
		FROM columns
		WHERE board_id = $1
		ORDER BY position ASC
	`, boardID); err != nil {
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

	boardID, err := repo.getCardBoardID(ctx, cardID)
	if err != nil {
		return domain.CardDetail{}, err
	}

	columns, err := repo.GetBoardColumns(ctx, boardID)
	if err != nil {
		return domain.CardDetail{}, err
	}

	todoColumnID, err := repo.getTodoColumnID(ctx, boardID)
	if err != nil {
		return domain.CardDetail{}, err
	}

	var children []domain.Card
	if err := repo.db.SelectContext(ctx, &children, `
		SELECT c.id, c.column_id, c.card_type, c.parent_card_id, c.title, c.description, c.position, c.created_at, c.archived, col.name AS column_name
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

	var projects []domain.Card
	var stories []domain.Card
	switch card.CardType {
	case domain.CardTypeStory:
		projects, err = repo.listBoardCardsByType(ctx, boardID, domain.CardTypeProject)
		if err != nil {
			return domain.CardDetail{}, err
		}
	case domain.CardTypeTask:
		stories, err = repo.listBoardCardsByType(ctx, boardID, domain.CardTypeStory)
		if err != nil {
			return domain.CardDetail{}, err
		}
	}

	return domain.CardDetail{
		Card:         card,
		BoardID:      boardID,
		ColumnName:   columnName,
		Columns:      columns,
		Children:     children,
		Projects:     projects,
		Stories:      stories,
		TodoColumnID: todoColumnID,
	}, nil
}

func (repo *Postgres) GetCard(ctx context.Context, cardID string) (domain.Card, error) {
	if _, err := repo.getCardBoardID(ctx, cardID); err != nil {
		return domain.Card{}, err
	}

	var card domain.Card
	if err := repo.db.GetContext(ctx, &card, `
		SELECT id, column_id, card_type, parent_card_id, title, description, position, created_at, archived
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

func (repo *Postgres) listBoardCardsByType(ctx context.Context, boardID string, cardType domain.CardType) ([]domain.Card, error) {
	var cards []domain.Card
	if err := repo.db.SelectContext(ctx, &cards, `
		SELECT c.id, c.column_id, c.card_type, c.parent_card_id, c.title, c.description, c.position, c.created_at, c.archived
		FROM cards c
		INNER JOIN columns col ON col.id = c.column_id
		WHERE col.board_id = $1 AND c.card_type = $2 AND c.archived = false
		ORDER BY c.title ASC, c.created_at ASC
	`, boardID, cardType); err != nil {
		return nil, fmt.Errorf("list board cards by type: %w", err)
	}
	if cards == nil {
		cards = []domain.Card{}
	}
	return cards, nil
}

func (repo *Postgres) UpdateCard(ctx context.Context, cardID, title, description, parentCardID string) (domain.Card, error) {
	card, err := repo.GetCard(ctx, cardID)
	if err != nil {
		return domain.Card{}, err
	}

	boardID, err := repo.getCardBoardID(ctx, cardID)
	if err != nil {
		return domain.Card{}, err
	}

	effectiveParentID := parentCardID
	if card.CardType == domain.CardTypeProject {
		effectiveParentID = ""
	} else {
		if err := repo.validateCardHierarchy(ctx, boardID, card.CardType, effectiveParentID); err != nil {
			return domain.Card{}, err
		}
		if effectiveParentID == cardID {
			return domain.Card{}, fmt.Errorf("card cannot be its own parent")
		}
		descendantIDs, err := repo.getDescendantCardIDs(ctx, cardID)
		if err != nil {
			return domain.Card{}, err
		}
		for _, descendantID := range descendantIDs {
			if descendantID == effectiveParentID {
				return domain.Card{}, fmt.Errorf("cannot set a descendant as parent")
			}
		}
	}

	_, err = repo.db.ExecContext(ctx, `
		UPDATE cards SET title = $1, description = $2, parent_card_id = $3 WHERE id = $4
	`, title, description, effectiveParentID, cardID)
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

	fromBoardID, err := repo.getColumnBoardID(ctx, fromColumnID)
	if err != nil {
		return err
	}
	toBoardID, err := repo.getColumnBoardID(ctx, toColumnID)
	if err != nil {
		return err
	}
	if fromBoardID != toBoardID {
		return fmt.Errorf("cannot move card to a different board")
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

func (repo *Postgres) ArchiveDoneInColumn(ctx context.Context, columnID string) error {
	var cardIDs []string
	if err := repo.db.SelectContext(ctx, &cardIDs, `
		SELECT id FROM cards WHERE column_id = $1 AND archived = false
	`, columnID); err != nil {
		return fmt.Errorf("get done cards: %w", err)
	}
	return repo.archiveCardsWithDescendants(ctx, cardIDs)
}

func (repo *Postgres) archiveCardsWithDescendants(ctx context.Context, rootCardIDs []string) error {
	if len(rootCardIDs) == 0 {
		return nil
	}

	archiveIDs := make(map[string]bool)
	for _, cardID := range rootCardIDs {
		archiveIDs[cardID] = true
		descendantIDs, err := repo.getDescendantCardIDs(ctx, cardID)
		if err != nil {
			return err
		}
		for _, descendantID := range descendantIDs {
			archiveIDs[descendantID] = true
		}
	}

	ids := make([]string, 0, len(archiveIDs))
	for cardID := range archiveIDs {
		ids = append(ids, cardID)
	}

	updateQuery, updateArgs, err := sqlx.In(`UPDATE cards SET archived = true WHERE id IN (?)`, ids)
	if err != nil {
		return fmt.Errorf("build archive query: %w", err)
	}
	updateQuery = repo.db.Rebind(updateQuery)

	if _, err := repo.db.ExecContext(ctx, updateQuery, updateArgs...); err != nil {
		return fmt.Errorf("archive cards: %w", err)
	}
	return nil
}

func (repo *Postgres) ListArchivedStories(ctx context.Context, boardID, query string, page int) (domain.ArchivedStoriesPage, error) {
	if err := repo.ensureBoardAccessible(ctx, boardID); err != nil {
		return domain.ArchivedStoriesPage{}, err
	}
	if page < 1 {
		page = 1
	}

	searchPattern := "%" + query + "%"
	var totalCount int
	countQuery := `
		SELECT COUNT(*)
		FROM cards c
		INNER JOIN columns col ON col.id = c.column_id
		WHERE col.board_id = $1
			AND c.card_type = 'story'
			AND c.archived = true
			AND ($2 = '' OR c.title ILIKE $3 OR c.description ILIKE $3)
	`
	if err := repo.db.GetContext(ctx, &totalCount, countQuery, boardID, query, searchPattern); err != nil {
		return domain.ArchivedStoriesPage{}, fmt.Errorf("count archived stories: %w", err)
	}

	totalPages := 1
	if totalCount > 0 {
		totalPages = (totalCount + archivedPageSize - 1) / archivedPageSize
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * archivedPageSize

	var stories []domain.Card
	listQuery := `
		SELECT c.id, c.column_id, c.card_type, c.parent_card_id, c.title, c.description, c.position, c.created_at, c.archived,
			col.name AS column_name, parent.title AS parent_title
		FROM cards c
		INNER JOIN columns col ON col.id = c.column_id
		LEFT JOIN cards parent ON parent.id::text = c.parent_card_id
		WHERE col.board_id = $1
			AND c.card_type = 'story'
			AND c.archived = true
			AND ($2 = '' OR c.title ILIKE $3 OR c.description ILIKE $3)
		ORDER BY c.created_at DESC
		LIMIT $4 OFFSET $5
	`
	if err := repo.db.SelectContext(ctx, &stories, listQuery, boardID, query, searchPattern, archivedPageSize, offset); err != nil {
		return domain.ArchivedStoriesPage{}, fmt.Errorf("list archived stories: %w", err)
	}
	if stories == nil {
		stories = []domain.Card{}
	}

	return domain.ArchivedStoriesPage{
		BoardID:    boardID,
		Stories:    stories,
		Query:      query,
		Page:       page,
		PageSize:   archivedPageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

func boardOrderClause(sortField, sortOrder string) string {
	order := "DESC"
	if sortOrder == "asc" {
		order = "ASC"
	}
	if sortField == "name" {
		return "name " + order
	}
	return "created_at " + order
}

func (repo *Postgres) ListFinishedBoards(ctx context.Context, query, sortField, sortOrder string, page int) (domain.FinishedBoardsPage, error) {
	if page < 1 {
		page = 1
	}
	if sortField != "name" {
		sortField = "created"
	}
	if sortOrder != "asc" {
		sortOrder = "desc"
	}

	searchPattern := "%" + query + "%"
	var totalCount int
	countQuery := `
		SELECT COUNT(*)
		FROM boards
		WHERE deleted = false
			AND finished = true
			AND ($1 = '' OR name ILIKE $2)
	`
	if err := repo.db.GetContext(ctx, &totalCount, countQuery, query, searchPattern); err != nil {
		return domain.FinishedBoardsPage{}, fmt.Errorf("count finished boards: %w", err)
	}

	totalPages := 1
	if totalCount > 0 {
		totalPages = (totalCount + archivedPageSize - 1) / archivedPageSize
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * archivedPageSize

	orderClause := boardOrderClause(sortField, sortOrder)
	listQuery := fmt.Sprintf(`
		SELECT id, name, created_at, finished, deleted
		FROM boards
		WHERE deleted = false
			AND finished = true
			AND ($1 = '' OR name ILIKE $2)
		ORDER BY %s
		LIMIT $3 OFFSET $4
	`, orderClause)

	var boards []domain.Board
	if err := repo.db.SelectContext(ctx, &boards, listQuery, query, searchPattern, archivedPageSize, offset); err != nil {
		return domain.FinishedBoardsPage{}, fmt.Errorf("list finished boards: %w", err)
	}
	if boards == nil {
		boards = []domain.Board{}
	}

	return domain.FinishedBoardsPage{
		Boards:     boards,
		Query:      query,
		SortField:  sortField,
		SortOrder:  sortOrder,
		Page:       page,
		PageSize:   archivedPageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}
