package domain

import "sort"

func PrepareColumnCards(cards []Card) []Card {
	if len(cards) == 0 {
		return cards
	}

	idsInColumn := make(map[string]bool, len(cards))
	for _, card := range cards {
		idsInColumn[card.ID] = true
	}

	childrenByParent := make(map[string][]Card)
	roots := make([]Card, 0)

	for _, card := range cards {
		if card.ParentCardID != "" && idsInColumn[card.ParentCardID] {
			childrenByParent[card.ParentCardID] = append(childrenByParent[card.ParentCardID], card)
			continue
		}
		roots = append(roots, card)
	}

	sort.Slice(roots, func(first, second int) bool {
		return roots[first].Position < roots[second].Position
	})

	for parentID := range childrenByParent {
		children := childrenByParent[parentID]
		sort.Slice(children, func(first, second int) bool {
			return children[first].Position < children[second].Position
		})
		childrenByParent[parentID] = children
	}

	result := make([]Card, 0, len(cards))
	var walk func(card Card, depth int, parentInColumn bool)
	walk = func(card Card, depth int, parentInColumn bool) {
		card.GroupDepth = depth
		card.ParentInSameColumn = parentInColumn
		result = append(result, card)

		for _, child := range childrenByParent[card.ID] {
			walk(child, depth+1, true)
		}
	}

	for _, root := range roots {
		walk(root, 0, false)
	}

	return result
}
