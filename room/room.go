package room

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/spf13/viper"
)

type Room struct {
	Config *viper.Viper
	Log    *slog.Logger

	Name string

	Players      map[string]*Player
	PlayersMutex sync.Mutex
}

var rooms map[string]*Room = make(map[string]*Room)

func NewRoomsMux() *http.ServeMux {
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
		Name:    name,
		Players: make(map[string]*Player),
	}

	rooms[name] = room

	return room, nil
}

func GetRoom(name string) *Room {
	return rooms[name]
}
