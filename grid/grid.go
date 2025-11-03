package grid

import (
	"fmt"
	"sort"
	"sync"

	"github.com/MatthewJM96/susnames/util"
)

type CardType uint

const (
	CIVILIAN CardType = iota
	SPY_TARGET
	COUNTERSPY_TARGET
)

type Card struct {
	Index    int
	Word     string
	Selected bool
	Type     CardType
	Votes    map[string]struct{}
}

type Grid struct {
	GridMutex sync.Mutex
	Cards     [25]*Card
}

func CreateGrid(spyCards int, counterspyCards int) *Grid {
	grid := &Grid{}

	// TODO(Matthew): support card decks, and even custom decks.
	// grid.assignCards()

	grid.assignTypes(spyCards, counterspyCards)

	return grid
}

func CreateGridFromWords(spyCards int, counterspyCards int, words [25]string) *Grid {
	var cards [25]*Card
	for i, word := range words {
		cards[i] = &Card{
			Index:    i,
			Word:     word,
			Selected: false,
			Type:     CIVILIAN,
			Votes:    make(map[string]struct{}),
		}
	}

	grid := &Grid{
		Cards: cards,
	}

	grid.assignTypes(spyCards, counterspyCards)

	return grid
}

func (g *Grid) assignTypes(spyCards int, counterspyCards int) {
	unsetCardTypes := 25
	util.RefreshRandSeed()

	for range spyCards {
		idx := util.Rnd.Intn(unsetCardTypes)
		for _, card := range g.Cards {
			if card.Type != CIVILIAN {
				continue
			}

			if idx == 0 {
				card.Type = SPY_TARGET
				break
			}

			idx -= 1
		}
		unsetCardTypes -= 1
	}

	for range counterspyCards {
		idx := util.Rnd.Intn(unsetCardTypes)
		for _, card := range g.Cards {
			if card.Type != CIVILIAN {
				continue
			}

			if idx == 0 {
				card.Type = COUNTERSPY_TARGET
				break
			}

			idx -= 1
		}
		unsetCardTypes -= 1
	}
}

func (g *Grid) ResetVote() {
	for _, card := range g.Cards {
		card.Votes = make(map[string]struct{})
	}
}

func (g *Grid) VoteCardAtIndex(index int, voteID string) (bool, *Card, error) {
	if index >= 25 {
		return false, nil, fmt.Errorf("card index %d out-of-range", index)
	}

	g.GridMutex.Lock()
	defer g.GridMutex.Unlock()

	card := g.Cards[index]

	if card.Selected {
		return false, nil, fmt.Errorf("card at index %d already selected", index)
	}

	_, exists := card.Votes[voteID]
	if exists {
		return false, card, nil
	}

	card.Votes[voteID] = struct{}{}

	return true, card, nil
}

func (g *Grid) UnvoteCardAtIndex(index int, voteID string) (bool, *Card, error) {
	if index >= 25 {
		return false, nil, fmt.Errorf("card index %d out-of-range", index)
	}

	g.GridMutex.Lock()
	defer g.GridMutex.Unlock()

	card := g.Cards[index]

	if card.Selected {
		return false, nil, fmt.Errorf("card at index %d already selected", index)
	}

	_, exists := card.Votes[voteID]
	if !exists {
		return false, card, nil
	}

	delete(card.Votes, voteID)

	return true, card, nil
}
func (g *Grid) EvaluateVote(suggestionCount int) (bool, error) {
	// Create a slice of cards with their vote counts
	cardsWithVotes := make([]*Card, 0)
	for _, card := range g.Cards {
		if len(card.Votes) > 0 {
			cardsWithVotes = append(cardsWithVotes, card)
		}
	}

	// Sort the cards by vote count in descending order
	sort.Slice(cardsWithVotes, func(i, j int) bool {
		return len(cardsWithVotes[i].Votes) > len(cardsWithVotes[j].Votes)
	})

	// Select the top N cards with the most votes
	selectedCards := cardsWithVotes[:min(suggestionCount, len(cardsWithVotes))]

	g.GridMutex.Lock()
	defer g.GridMutex.Unlock()

	// Mark the selected cards as selected
	for _, card := range selectedCards {
		card.Selected = true
	}

	return len(selectedCards) > 0, nil
}
