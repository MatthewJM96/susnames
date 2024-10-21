package room

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/MatthewJM96/susnames/util"
	"github.com/spf13/viper"
)

type Room struct {
	Config *viper.Viper
	Log    *slog.Logger

	Name string

	Players      map[string]*Player
	PlayersMutex sync.Mutex

	GameStateMutex sync.Mutex
	Started        bool
	Words          [25]string
}

var rooms map[string]*Room = make(map[string]*Room)

func GenerateRoomName() string {
	return util.GenerateRandomThreePartName()
}

func CreateRoom(config *viper.Viper, log *slog.Logger) (*Room, error) {
	var name string

	exists := true
	for range 5 {
		name = GenerateRoomName()

		_, exists = rooms[name]
	}
	if exists {
		return nil, fmt.Errorf("room name kept colliding, last tried: %s", name)
	}

	log.Info(fmt.Sprintf("created room: %s", name))

	room := &Room{
		Config:  config,
		Log:     log,
		Name:    name,
		Players: make(map[string]*Player),
		Started: false,
	}

	rooms[name] = room

	return room, nil
}

func GetRoom(name string) *Room {
	return rooms[name]
}

func (r *Room) StartGame(writer http.ResponseWriter, request *http.Request) {
	r.GameStateMutex.Lock()

	r.Started = true
	r.Words = [25]string{
		"relinquish", "genuine", "formula", "gain", "established", "development", "long",
		"personality", "package", "reveal", "premium", "carve", "authority", "blast",
		"compromise", "acid", "video", "live", "eject", "redundancy", "announcement",
		"tear", "depressed", "cunning", "child",
	}

	r.GameStateMutex.Unlock()

	r.BroadcastGrid(request.Context())
}

func (r *Room) Cookie(name string, value string) *http.Cookie {
	// Set cookies regarding a room to expire after 36 hours, that would be a long game
	// of susnames...
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Secure:   r.Config.GetBool("secure"),
		HttpOnly: r.Config.GetBool("http_only"),
		Expires:  time.Now().Add(36 * time.Hour),
		Path:     "/room/" + r.Name,
	}
}
