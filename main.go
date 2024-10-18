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

	router.Handle("/", middleware.NewSessionMiddleware(handlers.NewGreetHandler(config, log), config))
	router.Handle("/grid", middleware.NewSessionMiddleware(handlers.NewGridHandler(config, log), config))
	router.Handle("/api/set-name", middleware.NewSessionMiddleware(handlers.NewNameHandler(config, log), config))

	server := &http.Server{
		Addr:         "localhost:9000",
		Handler:      router,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	fmt.Printf("Listening on %v\n", server.Addr)
	server.ListenAndServe()
}
