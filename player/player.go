package player

import (
	"github.com/MatthewJM96/susnames/util"
)

type PlayerRole uint

const (
	SPECTATOR PlayerRole = iota
	SPYMASTER
	SPY
	COUNTERSPY
)

func GetPlayerRoleClass(role PlayerRole) string {
	if role == SPECTATOR {
		return "spectator"
	} else if role == SPYMASTER {
		return "spymaster"
	} else if role == SPY {
		return "spy"
	} else if role == COUNTERSPY {
		return "counterspy"
	}
	return ""
}

func GetViewablePlayerRoleClass(viewerRole PlayerRole, viewableRole PlayerRole) string {
	if viewableRole == SPECTATOR {
		return "spectator"
	} else if viewableRole == SPYMASTER {
		return "spymaster"
	} else if viewableRole == SPY {
		return "spy"
	} else if viewableRole == COUNTERSPY {
		if viewerRole == COUNTERSPY {
			return "counterspy"
		}

		return "spy"
	}
	return ""
}

type Player struct {
	SessionID string
	Name      string

	Role PlayerRole

	Votes int

	Msgs      chan []byte
	CloseConn func()
}

func GenerateRandomPlayerName() string {
	return util.GenerateRandomTwoPartName()
}

func NewPlayer(sessionID string, name string) *Player {
	return &Player{
		SessionID: sessionID,
		Name:      name,
		Role:      SPY,
		Votes:     0,
		Msgs:      make(chan []byte, 16),
	}
}
