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

	"github.com/MatthewJM96/susnames/grid"
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
	Spies          int // Note that this includes the number of counterspies.
	Counterspies   int
	Turn           PlayerRole
	Clue           string
	ClueMatches    int
	Grid           *grid.Grid
	VoteTimer      *time.Timer
	VoteEndVotes   int
	EndVotingOn    int
}

var rooms map[string]*Room = make(map[string]*Room)

const VOTE_TIME = 30 * time.Second

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
		Spies:        0,
		Counterspies: -1,
		EndVotingOn:  -1,
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
	r.Spies = 0
	for _, player := range r.Players {
		if player.Role == SPYMASTER {
			foundSpymaster = true
		} else if player.Role == SPY {
			r.Spies += 1
		}
	}

	/**
	 * Assign a default number of counterspies if none has been set.
	 */

	if r.Counterspies == -1 {
		r.Counterspies = int(math.Floor(float64(r.Spies-1) / 2.0))
	}

	util.RefreshRandSeed()

	/**
	 * Assign spymaster if no player has claimed the role.
	 */

	if !foundSpymaster {
		idx := util.Rnd.Intn(r.Spies)
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
		r.Spies -= 1
	}

	/**
	 * Assign counterspies.
	 */

	for range r.Counterspies {
		idx := util.Rnd.Intn(r.Spies)
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
		r.Spies -= 1
	}

	r.PlayersMutex.Unlock()
}

func (r *Room) startGame() {
	r.GameStateMutex.Lock()

	r.Started = true
	r.Turn = SPYMASTER
	r.Grid = grid.CreateGridFromWords(
		12,
		6,
		[25]string{
			"relinquish", "genuine", "formula", "gain", "established", "development",
			"long", "personality", "package", "reveal", "premium", "carve", "authority",
			"blast", "compromise", "acid", "video", "live", "eject", "redundancy",
			"announcement", "tear", "depressed", "cunning", "child",
		},
	)
	r.Clue = ""
	r.ClueMatches = 0
	r.VoteEndVotes = 0

	r.assignRoles()

	// TODO(Matthew): is this a satisfying way of doing this?
	if r.EndVotingOn == -1 {
		r.EndVotingOn = min(r.Counterspies+2, r.Spies)
	}

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

	r.VoteEndVotes += 1
	if r.VoteEndVotes >= r.EndVotingOn {
		r.Log.Info("voting closed by players")

		r.VoteTimer.Stop()

		go r.endVoting()
	} else {
		r.Log.Info(
			fmt.Sprintf(
				"(%s, %s) ended guessing, %d more to end vote",
				conn.Player.SessionID,
				conn.Player.Name,
				r.EndVotingOn-r.VoteEndVotes,
			),
		)
	}

	r.GameStateMutex.Unlock()
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

	r.Log.Info(
		fmt.Sprintf(
			"(%s, %s) suggested clue (%s, %d)",
			conn.Player.SessionID,
			conn.Player.Name,
			r.Clue,
			r.ClueMatches,
		),
	)

	r.VoteEndVotes = 0
	r.Grid.ResetVote()

	if r.VoteTimer != nil && r.VoteTimer.Stop() {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to suggest clue while vote timer was active",
				conn.Player.SessionID,
				conn.Player.Name,
			),
		)
	}

	r.Log.Info(fmt.Sprintf("voting open, ends in %s", VOTE_TIME.String()))

	r.VoteTimer = time.AfterFunc(
		VOTE_TIME,
		func() {
			r.Log.Info("voting closed by timeout")
			r.endVoting()
		},
	)

	r.GameStateMutex.Unlock()

	r.broadcastClue(context.Background())
}

func (r *Room) endVoting() {
	r.GameStateMutex.Lock()

	r.Grid.EvaluateVote()
	r.Turn = SPYMASTER

	r.GameStateMutex.Unlock()

	r.broadcastGameState(context.Background())
}

func (r *Room) voteCard(cardIndex int, conn *connectionManager) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if r.Turn != SPY {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to select a card while it wasn't the Spies' go",
				conn.Player.SessionID,
				conn.Player.Name,
			),
		)
		return
	}

	// TODO(Matthew): do we want the voting mechanism to allow for any spy or counterspy
	//	              to unilaterally select cards, or instead should it be timed voting
	//	              with "counters" on cards up to the number the spymaster presents?
	if conn.Player.Role != SPY && conn.Player.Role != COUNTERSPY {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to suggest clue but is not the Spymaster",
				conn.Player.SessionID,
				conn.Player.Name,
			),
		)
		return
	}

	_, err := r.Grid.VoteCardAtIndex(cardIndex)
	if err != nil {
		r.Log.Error(err.Error())
	}
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
	case "vote-card":
		cardIndex, err := strconv.Atoi(comm.Data0)
		if err != nil {
			r.Log.Error(fmt.Sprintf("could not parse Data0 as integer (card index): %s", comm.Data0))
			return
		}

		r.voteCard(cardIndex, conn)
	case "end-clue-guessing":
		r.endClueGuessing(conn)
	case "change-name":
		r.setPlayerName(comm.Data0)
	default:
		r.Log.Error(fmt.Sprintf("unrecognised command: %s", comm.Cmd))
	}
}
