package room

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/MatthewJM96/susnames/session"
	"github.com/coder/websocket"
	"github.com/goombaio/namegenerator"
)

type Player struct {
	SessionID string
	Name      string
	Room      *Room

	Msgs      chan []byte
	CloseConn func()
}

func GeneratePlayerName() string {
	seed := time.Now().UTC().UnixNano()
	nameGenerator := namegenerator.NewNameGenerator(seed)

	return nameGenerator.Generate()
}

func NewPlayer(session_id string, room *Room, msgs chan []byte, closeConn func()) *Player {
	return &Player{
		SessionID: session_id,
		Name:      GeneratePlayerName(),
		Room:      room,
		Msgs:      msgs,
		CloseConn: closeConn,
	}
}

/**
 * Creates a WebSocket connection to a player and associates them to this room. This
 * function then manages publishing messages to the player via the WebSocket connection.
 * Such messages can be queued via a channel stored with the player record.
 */
func (room *Room) ConnectPlayerToRoom(w http.ResponseWriter, r *http.Request) {
	session_id := session.GetSessionID(r)

	/**
	 * Obtain connection to websocket.
	 */
	var connectionMutex sync.Mutex
	connection, err := websocket.Accept(w, r, nil)
	if err != nil {
		room.Log.Error(err.Error())
		return
	}
	defer connection.CloseNow()

	/**
	 * Add player to room, establishing a means of closing connection from elsewhere.
	 */
	player, err := room.addPlayer(
		session_id,
		func() {
			connectionMutex.Lock()
			defer connectionMutex.Unlock()
			if connection != nil {
				connection.Close(
					websocket.StatusPolicyViolation,
					"connection too slow to keep up with messages",
				)
			}
		},
	)
	if err != nil {
		room.Log.Error(err.Error())
		return
	}
	defer room.removePlayer(session_id)

	/**
	 * Broadcast the existence of a new player in the room.
	 */
	room.BroadcastPlayerInfo(r.Context(), player)

	/**
	 * Until connection is closed keep publishing messages that we have in queue to
	 * player.
	 *
	 * TODO(Matthew): do we want to do something to not have  this spin so fast during
	 *				  inactivity?
	 */
	loop_ctx := connection.CloseRead(context.Background())
	for {
		select {
		case msg := <-player.Msgs:
			write_ctx, cancel := context.WithTimeout(loop_ctx, 5*time.Second)
			defer cancel()

			err = connection.Write(write_ctx, websocket.MessageText, msg)
			if err != nil {
				room.Log.Error(err.Error())
				return
			}
		case <-loop_ctx.Done():
			room.Log.Info(loop_ctx.Err().Error())
			return
		}
	}
}

func (room *Room) addPlayer(session_id string, closeConn func()) (*Player, error) {
	room.PlayersMutex.Lock()

	_, exists := room.Players[session_id]
	if exists {
		return nil, fmt.Errorf("player already exists with session ID: %s", session_id)
	}

	player := NewPlayer(
		session_id,
		room,
		make(chan []byte, 16),
		closeConn,
	)
	room.Players[session_id] = player

	room.Log.Info(fmt.Sprintf("added player: (%s, %s) to room %s", session_id, player.Name, room.Name))

	room.PlayersMutex.Unlock()

	return player, nil
}

func (room *Room) removePlayer(session_id string) error {
	room.PlayersMutex.Lock()

	player, exists := room.Players[session_id]
	if !exists {
		return fmt.Errorf("player with session ID does not exist to remove from room: %s", session_id)
	}

	room.Log.Info(fmt.Sprintf("removed player: (%s, %s) from room %s", session_id, player.Name, room.Name))

	delete(room.Players, session_id)

	room.PlayersMutex.Unlock()

	return nil
}

func (room *Room) GetPlayer(session_id string) (*Player, error) {
	player, exists := room.Players[session_id]
	if !exists || player == nil {
		return nil, fmt.Errorf("no player exists with session ID: %s", session_id)
	}

	return player, nil
}

func (room *Room) SetPlayerName(w http.ResponseWriter, r *http.Request) {
	session_id := session.GetSessionID(r)

	/**
	 * Get player to set name of.
	 */
	player, err := room.GetPlayer(session_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	/**
	 * Get name to set player to, generating one if we didn't get given one.
	 */
	name := ""
	if r.Form.Has("name") {
		name = r.FormValue("name")
	}
	if name == "" {
		name = GeneratePlayerName()
	}

	if player.Name == name {
		return
	}

	room.Log.Info(fmt.Sprintf("set player name: (%s, %s) to %s", session_id, player.Name, name))

	/**
	 * Set player name and broadcast the change.
	 */
	http.SetCookie(w, &http.Cookie{Name: "name", Value: name, Secure: room.Config.GetBool("secure"), HttpOnly: room.Config.GetBool("http_only")})

	player.Name = name

	room.BroadcastPlayerInfo(r.Context(), player)
}
