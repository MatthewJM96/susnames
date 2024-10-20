package handler

import (
	"fmt"
	"net/http"

	"github.com/MatthewJM96/susnames/components"
	"github.com/MatthewJM96/susnames/room"
)

func (h *Handler) CreateRoom(writer http.ResponseWriter, request *http.Request) {
	room, err := room.CreateRoom(h.Config, h.Log)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	components.Room(
		room.Name,
		components.Grid(
			[25]string{
				"relinquish", "genuine", "formula", "gain", "established", "development", "long",
				"personality", "package", "reveal", "premium", "carve", "authority", "blast",
				"compromise", "acid", "video", "live", "eject", "redundancy", "announcement",
				"tear", "depressed", "cunning", "child",
			},
		),
	).Render(
		request.Context(),
		writer,
	)
}

func (h *Handler) JoinRoom(writer http.ResponseWriter, request *http.Request) {
	request.ParseForm()

	if !request.Form.Has("name") {
		http.Error(writer, "must supply a name of room to join", http.StatusBadRequest)
	}

	roomName := request.FormValue("name")
	if roomName == "" {
		http.Error(writer, "must supply a name of room to join", http.StatusBadRequest)
	}

	room := room.GetRoom(roomName)
	if room == nil {
		http.Error(writer, fmt.Sprintf("no room exists with name: %s", roomName), http.StatusBadRequest)
		return
	}

	components.Room(
		room.Name,
		components.Grid(
			[25]string{
				"relinquish", "genuine", "formula", "gain", "established", "development", "long",
				"personality", "package", "reveal", "premium", "carve", "authority", "blast",
				"compromise", "acid", "video", "live", "eject", "redundancy", "announcement",
				"tear", "depressed", "cunning", "child",
			},
		),
	).Render(
		request.Context(),
		writer,
	)
}

func (h *Handler) ConnectPlayerToRoom(writer http.ResponseWriter, request *http.Request) {
	roomName := request.PathValue("name")

	room := room.GetRoom(roomName)
	if room == nil {
		http.Error(writer, fmt.Sprintf("no room exists with name: %s", roomName), http.StatusBadRequest)
		return
	}

	room.ConnectPlayerToRoom(writer, request)
}

func (h *Handler) SetPlayerName(writer http.ResponseWriter, request *http.Request) {
	roomName := request.PathValue("name")

	room := room.GetRoom(roomName)
	if room == nil {
		http.Error(writer, fmt.Sprintf("no room exists with name: %s", roomName), http.StatusBadRequest)
		return
	}

	room.SetPlayerName(writer, request)
}
