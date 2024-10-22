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

func (r *Room) makeGameState(ctx context.Context) []byte {
	buf := new(bytes.Buffer)

	components.Grid(r.Words).Render(ctx, buf)
	components.EmptyGameControl().Render(ctx, buf)

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
			return r.makeGameState(ctx), false
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

	r.broadcastMessageToPlayer(r.makeGameState(ctx), player)
}
