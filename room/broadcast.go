package room

import (
	"bytes"
	"context"

	"github.com/MatthewJM96/susnames/components"
	"github.com/MatthewJM96/susnames/grid"
	"github.com/MatthewJM96/susnames/player"
	"github.com/a-h/templ"
)

func (r *Room) broadcastMessage(messageFunc func(*player.Player) ([]byte, bool)) {
	r.PlayersMutex.Lock()
	defer r.PlayersMutex.Unlock()

	for _, player := range r.Players {
		message, skip := messageFunc(player)

		if skip {
			continue
		}

		select {
		case player.Msgs <- message:
		default:
			go player.CloseConn()
		}
	}
}

func (r *Room) broadcastMessageToPlayer(message []byte, p *player.Player) {
	select {
	case p.Msgs <- message:
	default:
		go p.CloseConn()
	}
}

func (r *Room) BroadcastPlayerList(ctx context.Context) {
	r.broadcastMessage(
		func(p *player.Player) ([]byte, bool) {
			buf := new(bytes.Buffer)

			tags := make([]templ.Component, 0, len(r.Players))

			tags = append(tags, components.PlayerNameTag(p.Name, player.GetPlayerRoleClass(p.Role)))

			for _, targetPlayer := range r.Players {
				if p == targetPlayer {
					continue
				}

				tags = append(
					tags,
					components.PlayerNameTag(
						targetPlayer.Name,
						player.GetViewablePlayerRoleClass(p.Role, targetPlayer.Role),
					),
				)
			}

			components.PlayerList(tags).Render(ctx, buf)

			return buf.Bytes(), false
		},
	)
}

func (r *Room) BroadcastCardToPlayer(ctx context.Context, p *player.Player, card *grid.Card) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if !r.Started {
		return
	}

	buf := new(bytes.Buffer)

	if p.Role == player.SPYMASTER {
		components.SpymasterCard(card).Render(ctx, buf)
	} else {
		components.SpyCard(
			card,
			card.Index,
			p.Role == player.COUNTERSPY,
			r.Turn == player.SPY,
			p.SessionID,
			p.Votes < r.ClueMatches+1,
		).Render(ctx, buf)
	}

	r.broadcastMessageToPlayer(buf.Bytes(), p)
}

func (r *Room) BroadcastGridToPlayer(ctx context.Context, p *player.Player, card *grid.Card) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if !r.Started {
		return
	}

	buf := new(bytes.Buffer)

	if p.Role == player.SPYMASTER {
		components.SpymasterGrid(r.Grid).Render(ctx, buf)
	} else {
		components.SpyGrid(
			r.Grid,
			p.Role == player.COUNTERSPY,
			r.Turn == player.SPY,
			p.SessionID,
			p.Votes < r.ClueMatches+1,
		).Render(ctx, buf)
	}

	r.broadcastMessageToPlayer(buf.Bytes(), p)
}

func (r *Room) makeGameState(ctx context.Context, p *player.Player) []byte {
	buf := new(bytes.Buffer)

	if p.Role == player.SPYMASTER {
		components.SpymasterGrid(r.Grid).Render(ctx, buf)
	} else {
		components.SpyGrid(
			r.Grid,
			p.Role == player.COUNTERSPY,
			r.Turn == player.SPY,
			p.SessionID,
			p.Votes < r.ClueMatches+1,
		).Render(ctx, buf)
	}
	components.EmptyGameControl().Render(ctx, buf)

	if r.Turn == player.SPYMASTER {
		if p.Role == player.SPYMASTER {
			components.ClueSuggestor().Render(ctx, buf)
		} else {
			components.EmptySpymasterSuggestion().Render(ctx, buf)
		}
	} else if r.Turn == player.SPY {
		if (p.Role == player.SPY || p.Role == player.COUNTERSPY) && !p.VotedEndGuessing {
			components.Clue(r.Clue, r.ClueMatches, true).Render(ctx, buf)
		} else {
			components.Clue(r.Clue, r.ClueMatches, false).Render(ctx, buf)
		}
	}

	return buf.Bytes()
}

func (r *Room) BroadcastGameState(ctx context.Context) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if !r.Started {
		return
	}

	r.broadcastMessage(
		func(p *player.Player) ([]byte, bool) {
			return r.makeGameState(ctx, p), false
		},
	)

	r.BroadcastPlayerList(ctx)
}

func (r *Room) BroadcastClue(ctx context.Context) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if !r.Started || r.Turn != player.SPY {
		return
	}

	r.broadcastMessage(
		func(p *player.Player) ([]byte, bool) {
			buf := new(bytes.Buffer)

			if (p.Role == player.SPY || p.Role == player.COUNTERSPY) && !p.VotedEndGuessing {
				components.Clue(r.Clue, r.ClueMatches, true).Render(ctx, buf)
			} else {
				components.Clue(r.Clue, r.ClueMatches, false).Render(ctx, buf)
			}

			return buf.Bytes(), false
		},
	)

	r.BroadcastPlayerList(ctx)
}

func (r *Room) BroadcastClueSuggestor(ctx context.Context) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if !r.Started || r.Turn != player.SPYMASTER {
		return
	}

	r.broadcastMessage(
		func(p *player.Player) ([]byte, bool) {

			buf := new(bytes.Buffer)

			if p.Role != player.SPYMASTER {
				components.EmptySpymasterSuggestion().Render(ctx, buf)
			} else {
				components.ClueSuggestor().Render(ctx, buf)
			}

			return buf.Bytes(), false
		},
	)

	r.BroadcastPlayerList(ctx)
}

func (r *Room) BroadcastGameStateToPlayer(ctx context.Context, p *player.Player) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if !r.Started {
		return
	}

	r.broadcastMessageToPlayer(r.makeGameState(ctx, p), p)
}

func (r *Room) BroadcastTimer(ctx context.Context) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if !r.Started {
		return
	}

	r.broadcastMessage(
		func(p *player.Player) ([]byte, bool) {
			buf := new(bytes.Buffer)

			if r.Turn == player.SPY && r.VoteTimer != nil {
				components.Timer(r.VoteTime).Render(ctx, buf)
			} else {
				components.NoTimer().Render(ctx, buf)
			}

			return buf.Bytes(), false
		},
	)

	r.BroadcastPlayerList(ctx)
}
