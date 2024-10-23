package grid

import (
	"errors"
	"fmt"
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
	Word     string
	Selected bool
	Type     CardType
	Votes    int
}

type Grid struct {
	GridMutex sync.Mutex
	Cards     [25]*Card
}

func CreateGrid(spyCards int, counterspyCards int) *Grid {
	grid := &Grid{}

	grid.assignTypes(spyCards, counterspyCards)

	return grid
}

func CreateGridFromWords(spyCards int, counterspyCards int, words [25]string) *Grid {
	var cards [25]*Card
	for i, word := range words {
		cards[i] = &Card{word, false, CIVILIAN, 0}
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
		card.Votes = 0
	}
}

func (g *Grid) VoteCardAtIndex(index int) (CardType, error) {
	if index >= 25 {
		return CIVILIAN, fmt.Errorf("card index %d out-of-range", index)
	}

	g.GridMutex.Lock()
	defer g.GridMutex.Unlock()

	card := g.Cards[index]

	if card.Selected {
		return card.Type, fmt.Errorf("card at index %d already selected", index)
	}

	card.Selected = true

	return card.Type, nil
}

func (g *Grid) VoteCardByWord(word string) (CardType, error) {
	g.GridMutex.Lock()
	defer g.GridMutex.Unlock()

	for index, card := range g.Cards {
		if card.Word == word {
			if card.Selected {
				return card.Type, fmt.Errorf("card at index %d already selected", index)
			}

			card.Selected = true

			return card.Type, nil
		}
	}

	return CIVILIAN, fmt.Errorf("no card with word %s", word)
}

func (g *Grid) EvaluateVote() error {
	highestVote := 0
	highestIndex := -1
	for index, card := range g.Cards {
		if card.Votes > highestVote {
			highestVote = card.Votes
			highestIndex = index
		}
	}

	if highestIndex == -1 {
		return errors.New("no card received a vote in voting round")
	}

	g.Cards[highestIndex].Selected = true

	return nil
}
