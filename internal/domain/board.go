package domain

import "time"

type CardType string

const (
	CardTypeProject CardType = "project"
	CardTypeStory   CardType = "story"
	CardTypeTask    CardType = "task"
)

type Board struct {
	ID           string    `db:"id"`
	Name         string    `db:"name"`
	CreatedAt    time.Time `db:"created_at"`
	Columns      []Column
	TodoColumnID string `db:"-"`
	Projects     []Card `db:"-"`
	Stories      []Card `db:"-"`
}

type Column struct {
	ID        string    `db:"id"`
	BoardID   string    `db:"board_id"`
	Name      string    `db:"name"`
	Position  int       `db:"position"`
	CreatedAt time.Time `db:"created_at"`
	Cards     []Card
}

type Card struct {
	ID                 string    `db:"id"`
	ColumnID           string    `db:"column_id"`
	CardType           CardType  `db:"card_type"`
	ParentCardID       string    `db:"parent_card_id"`
	Title              string    `db:"title"`
	Description        string    `db:"description"`
	Position           int       `db:"position"`
	CreatedAt          time.Time `db:"created_at"`
	ParentTitle        string    `db:"-"`
	ColumnName         string    `db:"column_name"`
	ParentInSameColumn bool      `db:"-"`
	GroupDepth         int       `db:"-"`
}

type CardDetail struct {
	Card
	ColumnName   string
	Columns      []Column
	Children     []Card
	TodoColumnID string
}

type StatusChangeResult struct {
	ModalDetail  CardDetail
	SourceColumn Column
	TargetColumn Column
	Moved        bool
}

type CreateCardResult struct {
	Card         Card
	Column       Column
	ModalDetail  CardDetail
	FromModal    bool
	TodoColumnID string
}

type DeleteCardResult struct {
	DeletedIDs []string
}

type CreateCardForm struct {
	Projects     []Card
	Stories      []Card
	TodoColumnID string
}

func (cardType CardType) IsValid() bool {
	switch cardType {
	case CardTypeProject, CardTypeStory, CardTypeTask:
		return true
	default:
		return false
	}
}

func (cardType CardType) Label() string {
	switch cardType {
	case CardTypeProject:
		return "Project"
	case CardTypeStory:
		return "Story"
	case CardTypeTask:
		return "Task"
	default:
		return string(cardType)
	}
}

func statusBadgeClass(columnName string) string {
	switch columnName {
	case "To Do":
		return "bg-slate-100 text-slate-700"
	case "In Progress":
		return "bg-amber-100 text-amber-800"
	case "Done":
		return "bg-emerald-100 text-emerald-800"
	default:
		return "bg-slate-100 text-slate-700"
	}
}

func statusBackgroundClass(columnName string) string {
	switch columnName {
	case "To Do":
		return "bg-slate-100"
	case "In Progress":
		return "bg-amber-50"
	case "Done":
		return "bg-emerald-50"
	default:
		return "bg-white"
	}
}

func (column Column) StatusBadgeClass() string {
	return statusBadgeClass(column.Name)
}

func (column Column) StatusBackgroundClass() string {
	return statusBackgroundClass(column.Name)
}

func (card Card) StatusBadgeClass() string {
	return statusBadgeClass(card.ColumnName)
}

func (detail CardDetail) StatusBadgeClass() string {
	return statusBadgeClass(detail.ColumnName)
}
