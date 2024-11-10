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
	"github.com/MatthewJM96/susnames/session"
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

func (r *Room) VoteEndClueGuessing(p *player.Player) {
	r.GameStateMutex.Lock()

	if r.Turn != player.SPY {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to stop guessing while it wasn't the Spies' go",
				p.SessionID,
				p.Name,
			),
		)
		return
	}

	if p.Role != player.SPY && p.Role != player.COUNTERSPY {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to stop guessing but is not a Spy",
				p.SessionID,
				p.Name,
			),
		)
		return
	}

	r.VoteEndVotes += 1
	if r.VoteEndVotes >= r.EndVotingOn || r.VoteEndVotes == r.Spies {
		r.Log.Info("voting closed by players")

		go r.EndVoting()
	} else {
		r.Log.Info(
			fmt.Sprintf(
				"(%s, %s) ended guessing, %d more to end vote",
				p.SessionID,
				p.Name,
				r.EndVotingOn-r.VoteEndVotes,
			),
		)
	}

	r.GameStateMutex.Unlock()
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

func (r *Room) EndVoting() {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if r.Turn != player.SPY {
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
	r.Turn = player.SPYMASTER

	go r.BroadcastGameState(context.Background())
	go r.BroadcastTimer(context.Background())
}

func (r *Room) VoteCard(cardIndex int, p *player.Player) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if r.Turn != player.SPY {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to vote for a card while it wasn't the Spies' go",
				p.SessionID,
				p.Name,
			),
		)
		return
	}

	if p.Role != player.SPY && p.Role != player.COUNTERSPY {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to vote for a card but is not a Spy or Counterspy",
				p.SessionID,
				p.Name,
			),
		)
		return
	}

	if p.Votes >= r.ClueMatches+1 {
		r.Log.Info(
			fmt.Sprintf(
				"(%s, %s) tried to vote for card %d but had hit max votes",
				p.SessionID,
				p.Name,
				cardIndex,
			),
		)
		return
	}

	voted, card, err := r.Grid.VoteCardAtIndex(cardIndex, p.SessionID)
	if err != nil {
		r.Log.Error(err.Error())
	}

	if voted {
		r.Log.Info(
			fmt.Sprintf(
				"(%s, %s) voted for card at index %d",
				p.SessionID,
				p.Name,
				cardIndex,
			),
		)

		if p.Votes == 0 {
			r.PlayersVoted += 1

			if r.PlayersVoted == r.VoteTimerAt {
				r.VoteTimer = time.AfterFunc(
					r.VoteTime,
					func() {
						r.Log.Info("voting closed by timeout")
						r.EndVoting()
					},
				)
				go r.BroadcastTimer(context.Background())
			}
		}

		p.Votes += 1

		go r.BroadcastCard(context.Background(), p, card)
	} else {
		r.Log.Warn(
			fmt.Sprintf(
				"(%s, %s) tried to vote for card at index %d but had already",
				p.SessionID,
				p.Name,
				cardIndex,
			),
		)
	}
}

func (r *Room) UnvoteCard(cardIndex int, p *player.Player) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if r.Turn != player.SPY {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to select a card while it wasn't the Spies' go",
				p.SessionID,
				p.Name,
			),
		)
		return
	}

	if p.Role != player.SPY && p.Role != player.COUNTERSPY {
		r.Log.Error(
			fmt.Sprintf(
				"(%s, %s) tried to suggest clue but is not the Spymaster",
				p.SessionID,
				p.Name,
			),
		)
		return
	}

	unvoted, card, err := r.Grid.UnvoteCardAtIndex(cardIndex, p.SessionID)
	if err != nil {
		r.Log.Error(err.Error())
	}

	if unvoted {
		r.Log.Info(
			fmt.Sprintf(
				"(%s, %s) unvoted card at index %d",
				p.SessionID,
				p.Name,
				cardIndex,
			),
		)

		p.Votes -= 1

		if p.Votes == 0 {
			r.PlayersVoted -= 1
		}

		go r.BroadcastCard(context.Background(), p, card)
	} else {
		r.Log.Warn(
			fmt.Sprintf(
				"(%s, %s) tried to unvote card at index %d but had not voted for it",
				p.SessionID,
				p.Name,
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

func (r *Room) AddPlayer(writer http.ResponseWriter, request *http.Request) (*player.Player, error) {
	r.PlayersMutex.Lock()
	defer r.PlayersMutex.Unlock()

	sessionID := session.SessionID()

	/**
	 * Ensure player has not yet connected.
	 */

	_, exists := r.Players[sessionID]
	if exists {
		return nil, fmt.Errorf("player already exists with session ID: %s", sessionID)
	}

	/**
	 * Obtain any existing name for player - maybe they've connected to the room before.
	 * If they have not, however, then generate an appropriate default.
	 */

	name := player.GenerateRandomPlayerName()
	cookie, err := request.Cookie("SN-Player-Name")
	if err == nil {
		name = cookie.Value
	} else {
		http.SetCookie(writer, r.cookie("SN-Player-Name", name))
	}

	player := player.NewPlayer(sessionID, name)
	r.Players[sessionID] = player

	r.Log.Info(fmt.Sprintf("added player: (%s, %s) to room %s", sessionID, player.Name, r.Name))

	return player, nil
}

func (r *Room) RemovePlayer(sessionID string) error {
	r.PlayersMutex.Lock()

	player, exists := r.Players[sessionID]
	if !exists {
		return fmt.Errorf("player with session ID does not exist to remove from room: %s", sessionID)
	}

	r.Log.Info(fmt.Sprintf("removed player: (%s, %s) from room %s", sessionID, player.Name, r.Name))

	delete(r.Players, sessionID)

	r.PlayersMutex.Unlock()

	r.BroadcastPlayerList(context.Background())

	return nil
}

func (r *Room) GetPlayer(sessionID string) (*player.Player, error) {
	player, exists := r.Players[sessionID]
	if !exists || player == nil {
		return nil, fmt.Errorf("no player exists with session ID: %s", sessionID)
	}

	return player, nil
}

func (r *Room) SetPlayerName(name string) {
	sessionID := session.SessionID()

	/**
	 * Get player to set name of.
	 */

	p, err := r.GetPlayer(sessionID)
	if err != nil {
		r.Log.Error(err.Error())
		return
	}

	/**
	 * Generare a player name if we weren't given one. If in any case the name is not
	 * to change, leave early.
	 */

	if name == "" {
		name = player.GenerateRandomPlayerName()
	}
	if p.Name == name {
		return
	}

	r.Log.Info(fmt.Sprintf("set player name: (%s, %s) to %s", sessionID, p.Name, name))

	/**
	 * Set player name and broadcast the change.
	 */

	p.Name = name

	r.BroadcastPlayerList(context.Background())
}
