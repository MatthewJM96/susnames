package room

import (
	"log/slog"
	"reflect"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

const MAX_MESSAGE_SIZE = 16384
const PING_PERIOD = 50 * time.Second
const PONG_WAIT = 10 * time.Second
const PONG_PERIOD = PING_PERIOD + PONG_WAIT
const WRITE_WAIT = 10 * time.Second
const MAX_MESSAGE_BATCH = 5

var upgrader = websocket.Upgrader{}

type connectionManager struct {
	Config *viper.Viper
	Log    *slog.Logger

	Conn   *websocket.Conn
	Room   *Room
	Player *Player
}

type command struct {
	Cmd  string `json:"cmd"`
	Data string `json:"data"`
}

func newConnectionManager(
	config *viper.Viper,
	log *slog.Logger,
	connection *websocket.Conn,
	room *Room,
	player *Player,
) *connectionManager {
	return &connectionManager{
		config,
		log,
		connection,
		room,
		player,
	}
}

func (c *connectionManager) readPump() {
	defer c.Conn.Close()
	defer c.Room.removePlayer(c.Player.SessionID)

	c.Conn.SetReadLimit(MAX_MESSAGE_SIZE)
	c.Conn.SetReadDeadline(time.Now().Add(PONG_PERIOD))
	c.Conn.SetPongHandler(
		func(string) error {
			c.Conn.SetReadDeadline(time.Now().Add(PONG_PERIOD))
			return nil
		},
	)

	for {
		comm := &command{}
		err := c.Conn.ReadJSON(comm)
		if err != nil {
			c.Log.Error(err.Error())
			break
		}

		c.Room.ProcessCommand(comm, c)
	}
}

func (c *connectionManager) writePump() {
	defer c.Conn.Close()

	ticker := time.NewTicker(PING_PERIOD)
	defer ticker.Stop()

	for {
		select {
		case message := <-c.Player.Msgs:
			c.Conn.SetWriteDeadline(time.Now().Add(WRITE_WAIT))

			/**
			 * Special handling of close request from somewhere in server.
			 */

			if reflect.DeepEqual(message, []byte("close")) {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			/**
			 * Establish writer and add the first message to the buffer.
			 */

			writer, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				c.Log.Error(err.Error())
				return
			}
			writer.Write(message)

			/**
			 * Close the writer, sending the buffered messages to the client.
			 */

			if err := writer.Close(); err != nil {
				c.Log.Error(err.Error())
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(WRITE_WAIT))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.Log.Error(err.Error())
				return
			}
		}
	}
}
