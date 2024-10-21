package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/MatthewJM96/susnames/handler"
	"github.com/MatthewJM96/susnames/session"
)

func main() {
	config := loadConfig()

	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	handlers := handler.NewHandler(config, log)

	router := http.NewServeMux()
	router.HandleFunc("GET /", handlers.Home)
	router.HandleFunc("POST /create-room", handlers.CreateRoom)
	router.HandleFunc("GET /room/{name}", handlers.ViewRoom)
	router.HandleFunc("POST /room/{name}", handlers.ViewRoom)
	router.HandleFunc("GET /room/{name}/conn", handlers.ConnectPlayerToRoom)
	router.HandleFunc("POST /room/{name}/name", handlers.SetPlayerName)
	router.HandleFunc("POST /room/{name}/start-game", handlers.StartGame)

	session := session.NewSessionMiddleware(router, config)

	server := &http.Server{
		Addr:         "localhost:9000",
		Handler:      session,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	fmt.Printf("Listening on %v\n", server.Addr)
	server.ListenAndServe()
}
