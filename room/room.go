package room

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
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
	Turn           PlayerRole
	Clue           string
	ClueMatches    int
	Words          [25]string
}

var rooms map[string]*Room = make(map[string]*Room)

func generateRoomName() string {
	return util.GenerateRandomThreePartName()
}

func CreateRoom(config *viper.Viper, log *slog.Logger) (*Room, error) {
	var name string

	exists := true
	for range 5 {
		name = generateRoomName()

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

func (r *Room) startGame() {
	r.GameStateMutex.Lock()

	r.Started = true
	r.Turn = SPYMASTER
	r.Words = [25]string{
		"relinquish", "genuine", "formula", "gain", "established", "development", "long",
		"personality", "package", "reveal", "premium", "carve", "authority", "blast",
		"compromise", "acid", "video", "live", "eject", "redundancy", "announcement",
		"tear", "depressed", "cunning", "child",
	}

	r.assignRoles()

	r.GameStateMutex.Unlock()

	r.broadcastGameState(context.Background())
}

func (r *Room) endClueGuessing(conn *connectionManager) {
	r.GameStateMutex.Lock()

	if r.Turn != SPY {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to stop guessing while it wasn't the Spies' go",
				conn.Player.SessionID,
				conn.Player.Name,
			),
		)
		return
	}

	if conn.Player.Role != SPY && conn.Player.Role != COUNTERSPY {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to stop guessing but is not a Spy",
				conn.Player.SessionID,
				conn.Player.Name,
			),
		)
		return
	}

	r.Turn = SPYMASTER

	r.Log.Info(fmt.Sprintf("(%s, %s) ended guessing", conn.Player.SessionID, conn.Player.Name))

	r.GameStateMutex.Unlock()

	r.broadcastClueSuggestor(context.Background())
}

func (r *Room) suggestClue(clue string, matches int, conn *connectionManager) {
	r.GameStateMutex.Lock()

	if r.Turn != SPYMASTER {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to suggest clue while it wasn't the Spymaster's go",
				conn.Player.SessionID,
				conn.Player.Name,
			),
		)
		return
	}

	if conn.Player.Role != SPYMASTER {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to suggest clue but is not the Spymaster",
				conn.Player.SessionID,
				conn.Player.Name,
			),
		)
		return
	}

	r.Turn = SPY
	r.Clue = clue
	r.ClueMatches = matches

	r.Log.Info(fmt.Sprintf("(%s, %s) suggested clue (%s, %d)", conn.Player.SessionID, conn.Player.Name, r.Clue, r.ClueMatches))

	r.GameStateMutex.Unlock()

	r.broadcastClue(context.Background())
}

func (r *Room) cookie(name string, value string) *http.Cookie {
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

func (r *Room) processCommand(comm *command, conn *connectionManager) {
	switch comm.Cmd {
	case "start-game":
		r.startGame()
	case "suggest-clue":
		clueMatches, err := strconv.Atoi(comm.Data1)
		if err != nil {
			r.Log.Error(fmt.Sprintf("could not parse Data1 as integer (clue matches): %s", comm.Data1))
			return
		}

		r.suggestClue(comm.Data0, clueMatches, conn)
	case "end-clue-guessing":
		r.endClueGuessing(conn)
	case "change-name":
		r.setPlayerName(comm.Data0)
	default:
		r.Log.Error(fmt.Sprintf("unrecognised command: %s", comm.Cmd))
	}
}
