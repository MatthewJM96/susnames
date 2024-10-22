package room

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MatthewJM96/susnames/session"
	"github.com/MatthewJM96/susnames/util"
)

type PlayerRole uint

const (
	SPECTATOR PlayerRole = iota
	SPYMASTER
	SPY
	COUNTERSPY
)

func getPlayerRoleClass(role PlayerRole) string {
	if role == SPECTATOR {
		return "spectator"
	} else if role == SPYMASTER {
		return "spymaster"
	} else if role == SPY {
		return "spy"
	} else if role == COUNTERSPY {
		return "counterspy"
	}
	return ""
}

func getPublicPlayerRoleClass(role PlayerRole) string {
	if role == SPECTATOR {
		return "spectator"
	} else if role == SPYMASTER {
		return "spymaster"
	} else if role == SPY {
		return "spy"
	} else if role == COUNTERSPY {
		return "spy"
	}
	return ""
}

type Player struct {
	SessionID string
	Name      string
	Room      *Room

	Role PlayerRole

	Msgs      chan []byte
	CloseConn func()
}

func generatePlayerName() string {
	return util.GenerateRandomTwoPartName()
}

func newPlayer(sessionID string, name string, room *Room) *Player {
	return &Player{
		SessionID: sessionID,
		Name:      name,
		Room:      room,
		Role:      SPY,
		Msgs:      make(chan []byte, 16),
	}
}

/**
 * Creates a WebSocket connection to a player and associates them to this room. This
 * function then manages publishing messages to the player via the WebSocket connection.
 * Such messages can be queued via a channel stored with the player record.
 */
func (r *Room) ConnectPlayerToRoom(writer http.ResponseWriter, request *http.Request) {
	sessionID := session.SessionID()

	/**
	 * Obtain any existing name for player - maybe they've connected to the room before.
	 */

	name := util.GenerateRandomTwoPartName()
	cookie, err := request.Cookie("SN-Player-Name")
	if err == nil {
		name = cookie.Value
	} else {
		http.SetCookie(writer, r.cookie("SN-Player-Name", name))
	}

	/**
	 * Add player to room.
	 */

	player, err := r.addPlayer(sessionID, name)
	if err != nil {
		r.Log.Error(err.Error())
		return
	}

	/**
	 * Obtain connection to websocket.
	 */

	connection, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		r.Log.Error(err.Error())
		return
	}

	r.Log.Info(fmt.Sprintf("websocket connection established with player (%s, %s)", sessionID, name))

	/**
	 * Set up read/write pumps to run until connection is closed.
	 */

	connManager := newConnectionManager(r.Config, r.Log, connection, r, player)
	go connManager.readPump()
	go connManager.writePump()

	/**
	 * broadcast the existence of a new player in the room, and if a game is ongoing,
	 * the state of that game.
	 */

	r.broadcastPlayerList(request.Context())

	if r.Started {
		r.broadcastGameStateToPlayer(request.Context(), player)
	}
}

func (r *Room) addPlayer(sessionID string, name string) (*Player, error) {
	r.PlayersMutex.Lock()

	_, exists := r.Players[sessionID]
	if exists {
		return nil, fmt.Errorf("player already exists with session ID: %s", sessionID)
	}

	player := newPlayer(sessionID, name, r)
	r.Players[sessionID] = player

	r.Log.Info(fmt.Sprintf("added player: (%s, %s) to room %s", sessionID, player.Name, r.Name))

	r.PlayersMutex.Unlock()

	return player, nil
}

func (r *Room) removePlayer(sessionID string) error {
	r.PlayersMutex.Lock()

	player, exists := r.Players[sessionID]
	if !exists {
		return fmt.Errorf("player with session ID does not exist to remove from room: %s", sessionID)
	}

	r.Log.Info(fmt.Sprintf("removed player: (%s, %s) from room %s", sessionID, player.Name, r.Name))

	delete(r.Players, sessionID)

	r.PlayersMutex.Unlock()

	r.broadcastPlayerList(context.Background())

	return nil
}

func (r *Room) getPlayer(sessionID string) (*Player, error) {
	player, exists := r.Players[sessionID]
	if !exists || player == nil {
		return nil, fmt.Errorf("no player exists with session ID: %s", sessionID)
	}

	return player, nil
}

func (r *Room) setPlayerName(name string) {
	sessionID := session.SessionID()

	/**
	 * Get player to set name of.
	 */

	player, err := r.getPlayer(sessionID)
	if err != nil {
		r.Log.Error(err.Error())
		return
	}

	/**
	 * Generare a player name if we weren't given one. If in any case the name is not
	 * to change, leave early.
	 */

	if name == "" {
		name = generatePlayerName()
	}
	if player.Name == name {
		return
	}

	r.Log.Info(fmt.Sprintf("set player name: (%s, %s) to %s", sessionID, player.Name, name))

	/**
	 * Set player name and broadcast the change.
	 */

	player.Name = name

	r.broadcastPlayerList(context.Background())
}
