package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/MatthewJM96/susnames/handlers"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	h := handlers.New(log)

	server := &http.Server{
		Addr:         "localhost:9000",
		Handler:      h,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	fmt.Printf("Listening on %v\n", server.Addr)
	server.ListenAndServe()
}
