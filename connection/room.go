package connection

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/MatthewJM96/susnames/room"
	"github.com/MatthewJM96/susnames/session"
)

/*
Creates a WebSocket connection to a player and associates them to this room. This
function then manages publishing messages to the player via the WebSocket connection.
Such messages can be queued via a channel stored with the player record.
*/
func ConnectPlayerToRoom(writer http.ResponseWriter, request *http.Request, r *room.Room) {
	/**
	 * Try to add player to room.
	 */

	p, err := r.AddPlayer(writer, request)
	if err != nil {
		r.Log.Error(err.Error())
		return
	}

	/**
	 * Obtain connection to websocket.
	 */

	connection, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		r.Log.Error(err.Error())
		return
	}

	r.Log.Info(fmt.Sprintf("websocket connection established with player (%s, %s)", session.SessionID(), p.Name))

	/**
	 * Set up read/write pumps to run until connection is closed.
	 */

	connManager := newConnectionManager(r.Config, r.Log, connection, r, p)
	go connManager.readPump()
	go connManager.writePump()

	/**
	 * broadcast the existence of a new player in the room, and if a game is ongoing,
	 * the state of that game.
	 */

	r.BroadcastPlayerList(request.Context())

	if r.Started {
		r.BroadcastGameStateToPlayer(request.Context(), p)
	}
}

func (c *connectionManager) processCommand(comm *command) {
	switch comm.Cmd {
	case "configure-game":
		counterspies, err := strconv.Atoi(comm.Data0)
		if err != nil {
			c.Room.Log.Error(fmt.Sprintf("could not parse Data0 as integer (counterspies): %s", comm.Data0))
			return
		}

		voteTimerDuration, err := strconv.Atoi(comm.Data1)
		if err != nil {
			c.Room.Log.Error(fmt.Sprintf("could not parse Data1 as integer (vote timer duration): %s", comm.Data1))
			return
		}

		voteTimerAt, err := strconv.Atoi(comm.Data2)
		if err != nil {
			c.Room.Log.Error(fmt.Sprintf("could not parse Data2 as integer (vote timer at): %s", comm.Data2))
			return
		}

		endVotingOn, err := strconv.Atoi(comm.Data3)
		if err != nil {
			c.Room.Log.Error(fmt.Sprintf("could not parse Data3 as integer (end voting after): %s", comm.Data3))
			return
		}

		c.Room.ConfigureGameSettings(counterspies, voteTimerDuration, voteTimerAt, endVotingOn)
	case "start-game":
		c.Room.StartGame()
	case "suggest-clue":
		clueMatches, err := strconv.Atoi(comm.Data1)
		if err != nil {
			c.Room.Log.Error(fmt.Sprintf("could not parse Data1 as integer (clue matches): %s", comm.Data1))
			return
		}

		c.Room.SuggestClue(comm.Data0, clueMatches, c.Player)
	case "vote-card":
		cardIndex, err := strconv.Atoi(comm.Data0)
		if err != nil {
			c.Room.Log.Error(fmt.Sprintf("could not parse Data0 as integer (card index): %s", comm.Data0))
			return
		}

		c.Room.VoteCard(cardIndex, c.Player)
	case "unvote-card":
		cardIndex, err := strconv.Atoi(comm.Data0)
		if err != nil {
			c.Room.Log.Error(fmt.Sprintf("could not parse Data0 as integer (card index): %s", comm.Data0))
			return
		}

		c.Room.UnvoteCard(cardIndex, c.Player)
	case "end-clue-guessing":
		c.Room.VoteEndClueGuessing(c.Player)
	case "change-name":
		c.Room.SetPlayerName(comm.Data0)
	default:
		c.Room.Log.Error(fmt.Sprintf("unrecognised command: %s", comm.Cmd))
	}
}
