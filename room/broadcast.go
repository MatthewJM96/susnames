package room

import (
	"bytes"
	"context"

	"github.com/MatthewJM96/susnames/components"
)

func (room *Room) BroadcastMessage(message []byte, exclude *Player) {
	room.PlayersMutex.Lock()
	defer room.PlayersMutex.Unlock()

	for _, player := range room.Players {
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

func (room *Room) BroadcastPlayerInfo(ctx context.Context, player *Player) {
	buf := new(bytes.Buffer)
	components.PlayerNameTag(player.SessionID, player.Name).Render(ctx, buf)

	room.BroadcastMessage(buf.Bytes(), player)
}
