package usecase

import (
	"context"

	"kanban/internal/domain"
	"kanban/internal/repository"
)

type Kanban struct {
	repo *repository.Postgres
}

func NewKanban(repo *repository.Postgres) *Kanban {
	return &Kanban{repo: repo}
}

func (uc *Kanban) ListBoards(ctx context.Context) ([]domain.Board, error) {
	return uc.repo.ListBoards(ctx)
}

func (uc *Kanban) CreateBoard(ctx context.Context, name string) (domain.Board, error) {
	return uc.repo.CreateBoard(ctx, name)
}

func (uc *Kanban) GetBoard(ctx context.Context, boardID string) (domain.Board, error) {
	return uc.repo.GetBoardWithColumnsAndCards(ctx, boardID)
}

func (uc *Kanban) GetColumn(ctx context.Context, columnID string) (domain.Column, error) {
	return uc.repo.GetColumnWithCards(ctx, columnID)
}

func (uc *Kanban) CreateCard(ctx context.Context, boardID string, cardType domain.CardType, parentCardID, title, description string) (domain.Card, error) {
	return uc.repo.CreateCard(ctx, boardID, cardType, parentCardID, title, description)
}

func (uc *Kanban) GetCard(ctx context.Context, cardID string) (domain.Card, error) {
	return uc.repo.GetCard(ctx, cardID)
}

func (uc *Kanban) GetCardDetail(ctx context.Context, cardID string) (domain.CardDetail, error) {
	return uc.repo.GetCardDetail(ctx, cardID)
}

func (uc *Kanban) UpdateCard(ctx context.Context, cardID, title, description string) (domain.Card, error) {
	return uc.repo.UpdateCard(ctx, cardID, title, description)
}

func (uc *Kanban) DeleteCard(ctx context.Context, cardID string) (domain.DeleteCardResult, error) {
	return uc.repo.DeleteCard(ctx, cardID)
}

func (uc *Kanban) MoveCard(ctx context.Context, cardID, toColumnID string, position int) error {
	return uc.repo.MoveCard(ctx, cardID, toColumnID, position)
}

func (uc *Kanban) ChangeCardStatus(ctx context.Context, cardID, columnID string) (domain.StatusChangeResult, error) {
	card, err := uc.repo.GetCard(ctx, cardID)
	if err != nil {
		return domain.StatusChangeResult{}, err
	}

	sourceColumnID := card.ColumnID
	moved := sourceColumnID != columnID

	if moved {
		targetColumn, err := uc.repo.GetColumnWithCards(ctx, columnID)
		if err != nil {
			return domain.StatusChangeResult{}, err
		}
		position := len(targetColumn.Cards)
		if err := uc.repo.MoveCard(ctx, cardID, columnID, position); err != nil {
			return domain.StatusChangeResult{}, err
		}
	}

	detail, err := uc.repo.GetCardDetail(ctx, cardID)
	if err != nil {
		return domain.StatusChangeResult{}, err
	}

	sourceColumn, err := uc.repo.GetColumnWithCards(ctx, sourceColumnID)
	if err != nil {
		return domain.StatusChangeResult{}, err
	}

	targetColumn := sourceColumn
	if moved {
		targetColumn, err = uc.repo.GetColumnWithCards(ctx, columnID)
		if err != nil {
			return domain.StatusChangeResult{}, err
		}
	}

	return domain.StatusChangeResult{
		ModalDetail:  detail,
		SourceColumn: sourceColumn,
		TargetColumn: targetColumn,
		Moved:        moved,
	}, nil
}

func (uc *Kanban) ArchiveDoneInColumn(ctx context.Context, columnID string) (domain.ArchiveDoneResult, error) {
	if err := uc.repo.ArchiveDoneInColumn(ctx, columnID); err != nil {
		return domain.ArchiveDoneResult{}, err
	}
	column, err := uc.repo.GetColumnWithCards(ctx, columnID)
	if err != nil {
		return domain.ArchiveDoneResult{}, err
	}
	return domain.ArchiveDoneResult{Column: column}, nil
}

func (uc *Kanban) ListArchivedStories(ctx context.Context, boardID, query string, page int) (domain.ArchivedStoriesPage, error) {
	return uc.repo.ListArchivedStories(ctx, boardID, query, page)
}
