package room

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MatthewJM96/susnames/player"
	"github.com/MatthewJM96/susnames/session"
)

func (r *Room) AddPlayer(writer http.ResponseWriter, request *http.Request) (*player.Player, error) {
	r.PlayersMutex.Lock()
	defer r.PlayersMutex.Unlock()

	sessionID := session.SessionID()

	/**
	 * Ensure player has not yet connected.
	 */

	_, exists := r.Players[sessionID]
	if exists {
		return nil, fmt.Errorf("player already exists with session ID: %s", sessionID)
	}

	/**
	 * Obtain any existing name for player - maybe they've connected to the room before.
	 * If they have not, however, then generate an appropriate default.
	 */

	name := player.GenerateRandomPlayerName()
	cookie, err := request.Cookie("SN-Player-Name")
	if err == nil {
		name = cookie.Value
	} else {
		http.SetCookie(writer, r.cookie("SN-Player-Name", name))
	}

	player := player.NewPlayer(sessionID, name)
	r.Players[sessionID] = player

	r.Log.Info(fmt.Sprintf("added player: (%s, %s) to room %s", sessionID, player.Name, r.Name))

	return player, nil
}

func (r *Room) RemovePlayer(sessionID string) error {
	r.PlayersMutex.Lock()

	player, exists := r.Players[sessionID]
	if !exists {
		return fmt.Errorf("player with session ID does not exist to remove from room: %s", sessionID)
	}

	r.Log.Info(fmt.Sprintf("removed player: (%s, %s) from room %s", sessionID, player.Name, r.Name))

	delete(r.Players, sessionID)

	r.PlayersMutex.Unlock()

	r.BroadcastPlayerList(context.Background())

	return nil
}

func (r *Room) GetPlayer(sessionID string) (*player.Player, error) {
	player, exists := r.Players[sessionID]
	if !exists || player == nil {
		return nil, fmt.Errorf("no player exists with session ID: %s", sessionID)
	}

	return player, nil
}

func (r *Room) SetPlayerName(name string) {
	sessionID := session.SessionID()

	/**
	 * Get player to set name of.
	 */

	p, err := r.GetPlayer(sessionID)
	if err != nil {
		r.Log.Error(err.Error())
		return
	}

	/**
	 * Generare a player name if we weren't given one. If in any case the name is not
	 * to change, leave early.
	 */

	if name == "" {
		name = player.GenerateRandomPlayerName()
	}
	if p.Name == name {
		return
	}

	r.Log.Info(fmt.Sprintf("set player name: (%s, %s) to %s", sessionID, p.Name, name))

	/**
	 * Set player name and broadcast the change.
	 */

	p.Name = name

	r.BroadcastPlayerList(context.Background())
}
