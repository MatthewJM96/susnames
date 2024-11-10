package room

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/MatthewJM96/susnames/grid"
	"github.com/MatthewJM96/susnames/player"
	"github.com/MatthewJM96/susnames/util"
	"github.com/spf13/viper"
)

type Room struct {
	Config *viper.Viper
	Log    *slog.Logger

	Name string

	Players      map[string]*player.Player
	PlayersMutex sync.Mutex

	GameStateMutex sync.Mutex
	Started        bool

	// Game state that is configurable by players.

	Spies        int // Note that this includes the number of counterspies.
	Counterspies int
	VoteTime     time.Duration
	/*
	  If 0, timer begins when voting begins, at any value from 1 to Room.Spies the
	  timer begins when that many spies/counterspies have ended guessing. Any other
	  value disables the timer.
	*/
	VoteTimerAt int
	/*
	  Indicates the number of votes to end voting required to actually end the voting
	  phase before the timeout. If this is set to a value <= 0 before starting a game,
	  then this value is set to 2 more than the number of counterspies (capped to the
	  total number of spies) at the start. Setting it to any value >= Room.Spies is
	  equivalent.
	*/
	EndVotingOn int

	// Game state that at most displayed to players.

	Turn         player.PlayerRole
	Clue         string
	ClueMatches  int
	Grid         *grid.Grid
	VoteTimer    *time.Timer
	VoteEndVotes int
	PlayersVoted int
}

var rooms map[string]*Room = make(map[string]*Room)

const DEFAULT_VOTE_TIME = 30 * time.Second

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
		Players:      make(map[string]*player.Player),
		Started:      false,
		Spies:        0,
		Counterspies: -1,
		VoteTime:     DEFAULT_VOTE_TIME,
		VoteTimerAt:  0,
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
	for _, p := range r.Players {
		if p.Role == player.SPYMASTER {
			foundSpymaster = true
		} else if p.Role == player.SPY {
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
		for _, p := range r.Players {
			if p.Role != player.SPY {
				continue
			}

			if idx == 0 {
				p.Role = player.SPYMASTER
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
		for _, p := range r.Players {
			if p.Role != player.SPY {
				continue
			}

			if idx == 0 {
				p.Role = player.COUNTERSPY
				break
			}

			idx -= 1
		}
		r.Spies -= 1
	}

	r.PlayersMutex.Unlock()
}

func (r *Room) StartGame() {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	r.Started = true
	r.Turn = player.SPYMASTER
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

	if r.EndVotingOn <= 0 {
		r.EndVotingOn = min(r.Counterspies+2, r.Spies)
	} else if r.EndVotingOn > r.Spies {
		r.EndVotingOn = r.Spies
	}

	go r.BroadcastGameState(context.Background())
}

func (r *Room) SuggestClue(clue string, matches int, p *player.Player) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if r.Turn != player.SPYMASTER {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to suggest clue while it wasn't the Spymaster's go",
				p.SessionID,
				p.Name,
			),
		)
		return
	}

	if p.Role != player.SPYMASTER {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to suggest clue but is not the Spymaster",
				p.SessionID,
				p.Name,
			),
		)
		return
	}

	r.Turn = player.SPY
	r.Clue = clue
	r.ClueMatches = matches

	r.Log.Info(
		fmt.Sprintf(
			"(%s, %s) suggested clue (%s, %d)",
			p.SessionID,
			p.Name,
			r.Clue,
			r.ClueMatches,
		),
	)

	r.Grid.ResetVote()

	r.VoteEndVotes = 0
	r.PlayersVoted = 0
	for _, player := range r.Players {
		player.Votes = 0
	}

	if r.VoteTimer != nil && r.VoteTimer.Stop() {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to suggest clue while vote timer was active",
				p.SessionID,
				p.Name,
			),
		)
	}

	r.Log.Info(fmt.Sprintf("voting open, ends in %s", r.VoteTime.String()))

	if r.VoteTimerAt == 0 {
		r.VoteTimer = time.AfterFunc(
			r.VoteTime,
			func() {
				r.Log.Info("voting closed by timeout")
				r.EndVoting()
			},
		)
		go r.BroadcastTimer(context.Background())
	}

	go r.BroadcastGameState(context.Background())
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
