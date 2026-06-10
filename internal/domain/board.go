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
	ParentTitle        string    `db:"parent_title"`
	ColumnName         string    `db:"column_name"`
	ParentInSameColumn bool      `db:"-"`
	GroupDepth         int       `db:"-"`
	ProjectCardID      string    `db:"-"`
	Archived           bool      `db:"archived"`
}

type BoardListPage struct {
	Boards []Board
}

type CardDetail struct {
	Card
	BoardID      string
	ColumnName   string
	Columns      []Column
	Children     []Card
	Projects     []Card
	Stories      []Card
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
	BoardID      string
	Projects     []Card
	Stories      []Card
	TodoColumnID string
}

type ArchivedStoriesPage struct {
	BoardID    string
	Stories    []Card
	Query      string
	Page       int
	PageSize   int
	TotalCount int
	TotalPages int
}

type ArchiveDoneResult struct {
	Column Column
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

func (card Card) IsDone() bool {
	return card.ColumnName == "Done"
}

func (page ArchivedStoriesPage) HasPreviousPage() bool {
	return page.Page > 1
}

func (page ArchivedStoriesPage) HasNextPage() bool {
	return page.Page < page.TotalPages
}

func (page ArchivedStoriesPage) PreviousPage() int {
	return page.Page - 1
}

func (page ArchivedStoriesPage) NextPage() int {
	return page.Page + 1
}

func (detail CardDetail) StatusBadgeClass() string {
	return statusBadgeClass(detail.ColumnName)
}

func BuildCardIndex(cards []Card) map[string]Card {
	index := make(map[string]Card, len(cards))
	for _, card := range cards {
		index[card.ID] = card
	}
	return index
}

func ResolveProjectCardID(card Card, cardIndex map[string]Card) string {
	switch card.CardType {
	case CardTypeProject:
		return card.ID
	case CardTypeStory:
		return card.ParentCardID
	case CardTypeTask:
		parentCard, found := cardIndex[card.ParentCardID]
		if !found {
			return ""
		}
		return parentCard.ParentCardID
	default:
		return ""
	}
}

func EnrichProjectCardIDs(cards []Card, cardIndex map[string]Card) {
	for index := range cards {
		cards[index].ProjectCardID = ResolveProjectCardID(cards[index], cardIndex)
	}
}
