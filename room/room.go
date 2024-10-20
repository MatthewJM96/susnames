package room

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/MatthewJM96/susnames/components"
	"github.com/MatthewJM96/susnames/session"
	"github.com/coder/websocket"
	"github.com/spf13/viper"
)

type Room struct {
	Config *viper.Viper
	Log    *slog.Logger

	Players      map[string]*Player
	PlayersMutex sync.Mutex
}

var rooms map[string]*Room = make(map[string]*Room)

func GetRoomsMux() *http.ServeMux {
	mux := http.NewServeMux()

	for name, room := range rooms {
		mux.HandleFunc("/room/"+name, room.ConnectPlayerToRoom)
		mux.HandleFunc("POST /room/"+name+"/name", room.SetPlayerName)
	}

	return mux
}

func CreateRoom(name string, config *viper.Viper, log *slog.Logger) (*Room, error) {
	_, exists := rooms[name]
	if exists {
		return nil, fmt.Errorf("room already exists with name: %s", name)
	}

	room := &Room{
		Config:  config,
		Log:     log,
		Players: make(map[string]*Player),
	}

	rooms[name] = room

	return room, nil
}

func GetRoom(name string) *Room {
	return rooms[name]
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

	room.PlayersMutex.Unlock()

	return player, nil
}

func (room *Room) removePlayer(session_id string) error {
	room.PlayersMutex.Lock()

	_, exists := room.Players[session_id]
	if !exists {
		return fmt.Errorf("player with session ID does not exist to remove from room: %s", session_id)
	}

	delete(room.Players, session_id)

	room.PlayersMutex.Unlock()

	return nil
}

func (room *Room) GetPlayer(session_id string) (*Player, error) {
	player, exists := room.Players[session_id]
	if !exists {
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

	room.Log.Info(fmt.Sprintf("set player name: %s", name))

	/**
	 * Set player name and broadcast the change.
	 */
	http.SetCookie(w, &http.Cookie{Name: "name", Value: name, Secure: room.Config.GetBool("secure"), HttpOnly: room.Config.GetBool("http_only")})

	player.Name = name

	room.BroadcastPlayerInfo(r.Context(), player)
}

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
