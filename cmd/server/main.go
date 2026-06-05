package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joao-carmo/blx/internal/handler"
	"github.com/joao-carmo/blx/internal/repository/ipac"
	"github.com/joao-carmo/blx/internal/service"
	"github.com/joao-carmo/blx/internal/web"
)

func main() {
	addr := os.Getenv("BLX_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	client := ipac.NewClient()
	repo := ipac.NewRepository(client)
	svc := service.NewCatalogService(repo)

	mux := http.NewServeMux()
	mux.Handle("/api/", handler.New(svc))
	mux.Handle("/", web.New(svc))

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("blx server listening on %s\n", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
