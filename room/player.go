package room

import (
	"time"

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
		session_id,
		GeneratePlayerName(),
		room,
		msgs,
		closeConn,
	}
}
