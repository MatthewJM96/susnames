package room

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/MatthewJM96/susnames/session"
	"github.com/MatthewJM96/susnames/util"
	"github.com/coder/websocket"
)

type Player struct {
	SessionID string
	Name      string
	Room      *Room

	Msgs      chan []byte
	CloseConn func()
}

func GeneratePlayerName() string {
	return util.GenerateRandomTwoPartName()
}

func NewPlayer(session_id string, name string, room *Room, msgs chan []byte, closeConn func()) *Player {
	return &Player{
		SessionID: session_id,
		Name:      name,
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
func (r *Room) ConnectPlayerToRoom(writer http.ResponseWriter, request *http.Request) {
	session_id := session.GetSessionID(request)

	/**
	 * Obtain any existing name for player - maybe they've connected to the room before.
	 */

	name := util.GenerateRandomTwoPartName()
	cookie, err := request.Cookie("SN-Player-Name")
	if err == nil {
		name = cookie.Value
	} else {
		http.SetCookie(writer, r.Cookie("SN-Player-Name", name))
	}

	/**
	 * Obtain connection to websocket.
	 */
	var connectionMutex sync.Mutex
	connection, err := websocket.Accept(writer, request, nil)
	if err != nil {
		r.Log.Error(err.Error())
		return
	}
	defer connection.CloseNow()

	/**
	 * Add player to room, establishing a means of closing connection from elsewhere.
	 */

	player, err := r.addPlayer(
		session_id,
		name,
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
		r.Log.Error(err.Error())
		return
	}
	defer r.removePlayer(session_id)

	/**
	 * Broadcast the existence of a new player in the room.
	 */
	r.BroadcastPlayerList(request.Context())

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
				r.Log.Error(err.Error())
				return
			}
		case <-loop_ctx.Done():
			r.Log.Info(loop_ctx.Err().Error())
			return
		}
	}
}

func (r *Room) addPlayer(session_id string, name string, closeConn func()) (*Player, error) {
	r.PlayersMutex.Lock()

	_, exists := r.Players[session_id]
	if exists {
		return nil, fmt.Errorf("player already exists with session ID: %s", session_id)
	}

	player := NewPlayer(
		session_id,
		name,
		r,
		make(chan []byte, 16),
		closeConn,
	)
	r.Players[session_id] = player

	r.Log.Info(fmt.Sprintf("added player: (%s, %s) to room %s", session_id, player.Name, r.Name))

	r.PlayersMutex.Unlock()

	return player, nil
}

func (r *Room) removePlayer(session_id string) error {
	r.PlayersMutex.Lock()

	player, exists := r.Players[session_id]
	if !exists {
		return fmt.Errorf("player with session ID does not exist to remove from room: %s", session_id)
	}

	r.Log.Info(fmt.Sprintf("removed player: (%s, %s) from room %s", session_id, player.Name, r.Name))

	delete(r.Players, session_id)

	r.PlayersMutex.Unlock()

	r.BroadcastPlayerList(context.Background())

	return nil
}

func (r *Room) GetPlayer(session_id string) (*Player, error) {
	player, exists := r.Players[session_id]
	if !exists || player == nil {
		return nil, fmt.Errorf("no player exists with session ID: %s", session_id)
	}

	return player, nil
}

func (r *Room) SetPlayerName(writer http.ResponseWriter, request *http.Request) {
	session_id := session.GetSessionID(request)

	/**
	 * Get player to set name of.
	 */
	player, err := r.GetPlayer(session_id)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	/**
	 * Get name to set player to, generating one if we didn't get given one.
	 */
	request.ParseForm()

	name := ""
	if request.Form.Has("name") {
		name = request.FormValue("name")
	}
	if name == "" {
		name = GeneratePlayerName()
	}

	if player.Name == name {
		return
	}

	r.Log.Info(fmt.Sprintf("set player name: (%s, %s) to %s", session_id, player.Name, name))

	/**
	 * Set player name and broadcast the change.
	 */
	http.SetCookie(writer, &http.Cookie{Name: "name", Value: name, Secure: r.Config.GetBool("secure"), HttpOnly: r.Config.GetBool("http_only")})

	player.Name = name

	r.BroadcastPlayerList(request.Context())
}
