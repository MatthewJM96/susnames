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

	Turn         PlayerRole
	Clue         string
	ClueMatches  int
	Grid         *grid.Grid
	VoteTimer    *time.Timer
	VoteEndVotes int
	PlayersVoted int
}

var rooms map[string]*Room = make(map[string]*Room)

const DEFAULT_VOTE_TIME = 300 * time.Second

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
	defer r.GameStateMutex.Unlock()

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

	if r.EndVotingOn <= 0 {
		r.EndVotingOn = min(r.Counterspies+2, r.Spies)
	} else if r.EndVotingOn > r.Spies {
		r.EndVotingOn = r.Spies
	}

	go r.broadcastGameState(context.Background())
}

func (r *Room) voteEndClueGuessing(conn *connectionManager) {
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
	if r.VoteEndVotes >= r.EndVotingOn || r.VoteEndVotes == r.Spies {
		r.Log.Info("voting closed by players")

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
	defer r.GameStateMutex.Unlock()

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
				conn.Player.SessionID,
				conn.Player.Name,
			),
		)
	}

	r.Log.Info(fmt.Sprintf("voting open, ends in %s", r.VoteTime.String()))

	if r.VoteTimerAt == 0 {
		r.VoteTimer = time.AfterFunc(
			r.VoteTime,
			func() {
				r.Log.Info("voting closed by timeout")
				r.endVoting()
			},
		)
		go r.broadcastTimer(context.Background())
	}

	go r.broadcastGameState(context.Background())
}

func (r *Room) endVoting() {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if r.Turn != SPY {
		r.GameStateMutex.Unlock()
		return
	}

	// Call again as while it will often do nothing, if the timer was to start at the
	// very moment voting was voted to end (a.k.a. majority agreed the clue couldn't
	// lead to voting for any cards, but someone at that moment voted for a card), then
	// there could be a race condition reaching this function.
	r.VoteTimer.Stop()

	r.VoteTimer = nil

	r.Grid.EvaluateVote()
	r.Turn = SPYMASTER

	go r.broadcastGameState(context.Background())
	go r.broadcastTimer(context.Background())
}

func (r *Room) voteCard(cardIndex int, conn *connectionManager) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if r.Turn != SPY {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to vote for a card while it wasn't the Spies' go",
				conn.Player.SessionID,
				conn.Player.Name,
			),
		)
		return
	}

	if conn.Player.Role != SPY && conn.Player.Role != COUNTERSPY {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to vote for a card but is not a Spy or Counterspy",
				conn.Player.SessionID,
				conn.Player.Name,
			),
		)
		return
	}

	if conn.Player.Votes >= r.ClueMatches+1 {
		r.Log.Info(
			fmt.Sprintf(
				"(%s, %s) tried to vote for card %d but had hit max votes",
				conn.Player.SessionID,
				conn.Player.Name,
				cardIndex,
			),
		)
		return
	}

	voted, card, err := r.Grid.VoteCardAtIndex(cardIndex, conn.Player.SessionID)
	if err != nil {
		r.Log.Error(err.Error())
	}

	if voted {
		r.Log.Info(
			fmt.Sprintf(
				"(%s, %s) voted for card at index %d",
				conn.Player.SessionID,
				conn.Player.Name,
				cardIndex,
			),
		)

		if conn.Player.Votes == 0 {
			r.PlayersVoted += 1

			if r.PlayersVoted == r.VoteTimerAt {
				r.VoteTimer = time.AfterFunc(
					r.VoteTime,
					func() {
						r.Log.Info("voting closed by timeout")
						r.endVoting()
					},
				)
				go r.broadcastTimer(context.Background())
			}
		}

		conn.Player.Votes += 1

		go r.broadcastCard(context.Background(), conn.Player, card)
	} else {
		r.Log.Warn(
			fmt.Sprintf(
				"(%s, %s) tried to vote for card at index %d but had already",
				conn.Player.SessionID,
				conn.Player.Name,
				cardIndex,
			),
		)
	}
}

func (r *Room) unvoteCard(cardIndex int, conn *connectionManager) {
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

	unvoted, card, err := r.Grid.UnvoteCardAtIndex(cardIndex, conn.Player.SessionID)
	if err != nil {
		r.Log.Error(err.Error())
	}

	if unvoted {
		r.Log.Info(
			fmt.Sprintf(
				"(%s, %s) unvoted card at index %d",
				conn.Player.SessionID,
				conn.Player.Name,
				cardIndex,
			),
		)

		conn.Player.Votes -= 1

		if conn.Player.Votes == 0 {
			r.PlayersVoted -= 1
		}

		go r.broadcastCard(context.Background(), conn.Player, card)
	} else {
		r.Log.Warn(
			fmt.Sprintf(
				"(%s, %s) tried to unvote card at index %d but had not voted for it",
				conn.Player.SessionID,
				conn.Player.Name,
				cardIndex,
			),
		)
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
	case "unvote-card":
		cardIndex, err := strconv.Atoi(comm.Data0)
		if err != nil {
			r.Log.Error(fmt.Sprintf("could not parse Data0 as integer (card index): %s", comm.Data0))
			return
		}

		r.unvoteCard(cardIndex, conn)
	case "end-clue-guessing":
		r.voteEndClueGuessing(conn)
	case "change-name":
		r.setPlayerName(comm.Data0)
	default:
		r.Log.Error(fmt.Sprintf("unrecognised command: %s", comm.Cmd))
	}
}
