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
)

func main() {
	addr := os.Getenv("BLX_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	client := ipac.NewClient()
	repo := ipac.NewRepository(client)
	svc := service.NewCatalogService(repo)
	h := handler.New(svc)

	server := &http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("blx server listening on %s\n", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
