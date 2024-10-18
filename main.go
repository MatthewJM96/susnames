package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/MatthewJM96/susnames/handlers"
	"github.com/MatthewJM96/susnames/middleware"
)

func main() {
	config := loadConfig()

	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	router := http.NewServeMux()

	router.Handle("/", handlers.NewGreetHandler(config, log))
	router.Handle("/grid", handlers.NewGridHandler(config, log))
	router.Handle("/api/name", handlers.NewNameHandler(config, log))

	session := middleware.NewSessionMiddleware(router, config)

	server := &http.Server{
		Addr:         "localhost:9000",
		Handler:      session,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	fmt.Printf("Listening on %v\n", server.Addr)
	server.ListenAndServe()
}
