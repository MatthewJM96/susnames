package room

import (
	"bytes"
	"context"

	"github.com/MatthewJM96/susnames/components"
	"github.com/a-h/templ"
)

func (r *Room) broadcastMessage(messageFunc func(*Player) ([]byte, bool)) {
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

func (r *Room) broadcastMessageToPlayer(message []byte, player *Player) {
	select {
	case player.Msgs <- message:
	default:
		go player.CloseConn()
	}
}

func (r *Room) broadcastPlayerList(ctx context.Context) {
	r.broadcastMessage(
		func(player *Player) ([]byte, bool) {
			buf := new(bytes.Buffer)

			tags := make([]templ.Component, 0, len(r.Players))

			tags = append(tags, components.PlayerNameTag(player.Name, getPlayerRoleClass(player.Role)))

			for _, targetPlayer := range r.Players {
				if player == targetPlayer {
					continue
				}

				tags = append(tags, components.PlayerNameTag(targetPlayer.Name, getPublicPlayerRoleClass(targetPlayer.Role)))
			}

			components.PlayerList(tags).Render(ctx, buf)

			return buf.Bytes(), false
		},
	)
}

func (r *Room) makeGameState(ctx context.Context, player *Player) []byte {
	buf := new(bytes.Buffer)

	components.Grid(r.Grid).Render(ctx, buf)
	components.EmptyGameControl().Render(ctx, buf)

	if r.Turn == SPYMASTER && player.Role == SPYMASTER {
		components.ClueSuggestor().Render(ctx, buf)
	} else if r.Turn == SPY {
		if player.Role == SPY || player.Role == COUNTERSPY {
			components.Clue(r.Clue, r.ClueMatches, true).Render(ctx, buf)
		} else {
			components.Clue(r.Clue, r.ClueMatches, false).Render(ctx, buf)
		}
	}

	return buf.Bytes()
}

func (r *Room) broadcastGameState(ctx context.Context) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if !r.Started {
		return
	}

	r.broadcastMessage(
		func(player *Player) ([]byte, bool) {
			return r.makeGameState(ctx, player), false
		},
	)

	r.broadcastPlayerList(ctx)
}

func (r *Room) broadcastClue(ctx context.Context) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if !r.Started || r.Turn != SPY {
		return
	}

	r.broadcastMessage(
		func(p *Player) ([]byte, bool) {
			buf := new(bytes.Buffer)

			if p.Role == SPY || p.Role == COUNTERSPY {
				components.Clue(r.Clue, r.ClueMatches, true).Render(ctx, buf)
			} else {
				components.Clue(r.Clue, r.ClueMatches, false).Render(ctx, buf)
			}

			return buf.Bytes(), false
		},
	)

	r.broadcastPlayerList(ctx)
}

func (r *Room) broadcastClueSuggestor(ctx context.Context) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if !r.Started || r.Turn != SPYMASTER {
		return
	}

	r.broadcastMessage(
		func(player *Player) ([]byte, bool) {

			buf := new(bytes.Buffer)

			if player.Role != SPYMASTER {
				components.EmptySpymasterSuggestion().Render(ctx, buf)
			} else {
				components.ClueSuggestor().Render(ctx, buf)
			}

			return buf.Bytes(), false
		},
	)

	r.broadcastPlayerList(ctx)
}

func (r *Room) broadcastGameStateToPlayer(ctx context.Context, player *Player) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if !r.Started {
		return
	}

	r.broadcastMessageToPlayer(r.makeGameState(ctx, player), player)
}
