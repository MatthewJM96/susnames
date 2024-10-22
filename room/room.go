package room

import (
	"fmt"
	"log/slog"
	"math"
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
	Counterspies   int
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
		Config:       config,
		Log:          log,
		Name:         name,
		Players:      make(map[string]*Player),
		Started:      false,
		Counterspies: -1,
	}

	rooms[name] = room

	return room, nil
}

func GetRoom(name string) *Room {
	return rooms[name]
}

func (r *Room) assignRoles() {
	r.PlayersMutex.Lock()

	/**
	 * Look for spymaster, and count number of players who will be playing.
	 */

	foundSpymaster := false
	spies := 0
	for _, player := range r.Players {
		if player.Role == SPYMASTER {
			foundSpymaster = true
		} else if player.Role == SPY {
			spies += 1
		}
	}

	/**
	 * Assign a default number of counterspies if none has been set.
	 */

	if r.Counterspies == -1 {
		r.Counterspies = int(math.Floor(float64(spies-1) / 2.0))
	}

	util.RefreshRandSeed()

	/**
	 * Assign spymaster if no player has claimed the role.
	 */

	if !foundSpymaster {
		idx := util.Rnd.Intn(spies)
		for _, player := range r.Players {
			if player.Role != SPY {
				continue
			}

			if idx == 0 {
				player.Role = SPYMASTER
				break
			}

			idx -= 1
		}
		spies -= 1
	}

	/**
	 * Assign counterspies.
	 */

	for range r.Counterspies {
		idx := util.Rnd.Intn(spies)
		for _, player := range r.Players {
			if player.Role != SPY {
				continue
			}

			if idx == 0 {
				player.Role = COUNTERSPY
				break
			}

			idx -= 1
		}
		spies -= 1
	}

	r.PlayersMutex.Unlock()
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

	r.assignRoles()

	r.GameStateMutex.Unlock()

	r.BroadcastGameState(request.Context())
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

func (r *Room) ProcessCommand(comm *command, conn *connectionManager) {
	switch comm.Cmd {
	case "change-name":
		r.SetPlayerName(comm.Data)
	default:
		r.Log.Error(fmt.Sprintf("unrecognised command: %s", comm.Cmd))
	}
}
