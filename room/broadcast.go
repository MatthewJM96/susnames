package room

import (
	"bytes"
	"context"

	"github.com/MatthewJM96/susnames/components"
)

func (r *Room) BroadcastMessage(message []byte, exclude *Player) {
	r.PlayersMutex.Lock()
	defer r.PlayersMutex.Unlock()

	for _, player := range r.Players {
		if player == exclude {
			continue
		}

		select {
		case player.Msgs <- message:
		default:
			go player.CloseConn()
		}
	}
}

func (r *Room) BroadcastPlayerInfo(ctx context.Context, player *Player) {
	buf := new(bytes.Buffer)
	components.PlayerNameTag(player.SessionID, player.Name).Render(ctx, buf)

	r.BroadcastMessage(buf.Bytes(), player)
}
