package room

import (
	"context"
	"fmt"
	"time"

	"github.com/MatthewJM96/susnames/player"
)

func (r *Room) VoteEndClueGuessing(p *player.Player) {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

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

	p.PlayerStateMutex.Lock()
	defer p.PlayerStateMutex.Unlock()

	if !p.VotedEndGuessing {
		p.VotedEndGuessing = true
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

			// This is okay despite broadcasting to all players as we next intend to
			// broadcast the number that have ended guessing.
			go r.BroadcastClue(context.Background())
		}
	} else {
		r.Log.Info(
			fmt.Sprintf(
				"(%s, %s) tried to end guessing, but had already ended guessing",
				p.SessionID,
				p.Name,
			),
		)
	}
}

func (r *Room) EndVoting() {
	r.GameStateMutex.Lock()
	defer r.GameStateMutex.Unlock()

	if r.Turn != player.SPY {
		return
	}

	if r.VoteTimer != nil {
		// Call again as while it will often do nothing, if the timer was to start at the
		// very moment voting was voted to end (a.k.a. majority agreed the clue couldn't
		// lead to voting for any cards, but someone at that moment voted for a card), then
		// there could be a race condition reaching this function.
		r.VoteTimer.Stop()

		r.VoteTimer = nil
	}

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
				r.Log.Info(fmt.Sprintf("voting open, ends in %s", r.VoteTime.String()))

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

		p.PlayerStateMutex.Lock()
		defer p.PlayerStateMutex.Unlock()

		p.Votes += 1

		/**
		 * Broadcast whole grid if player has voted with their last vote (need to disable
		 * their voting capability in that case).
		 */
		if p.Votes >= r.ClueMatches+1 {
			go r.BroadcastGridToPlayer(context.Background(), p, card)
		} else {
			go r.BroadcastCardToPlayer(context.Background(), p, card)
		}
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

		p.PlayerStateMutex.Lock()
		defer p.PlayerStateMutex.Unlock()

		p.Votes -= 1

		if p.Votes == 0 {
			r.PlayersVoted -= 1
		}

		/**
		 * Broadcast whole grid if player has unvoted their last vote (need to reenable
		 * their voting capability in that case).
		 */
		if p.Votes == r.ClueMatches {
			go r.BroadcastGridToPlayer(context.Background(), p, card)
		} else {
			go r.BroadcastCardToPlayer(context.Background(), p, card)
		}
		go r.BroadcastGridToPlayer(context.Background(), p, card)
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
