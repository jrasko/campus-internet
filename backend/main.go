package main

import (
	"backend/api"
	"backend/model"
	"backend/repository"
	"backend/service"
	"backend/service/allocation"
	"backend/service/confwriter"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

type application struct {
	service    api.MemberService
	repository service.MemberRepository
	router     http.Handler
}

func main() {
	config, err := model.LoadConfig(context.Background())
	if err != nil {
		panic(fmt.Errorf("when resolving config: %w", err))
	}
	log.Println("[INFO] Loaded config")

	app, err := newApplication(config)
	if err != nil {
		panic(err)
	}

	app.start()
}

func newApplication(config model.Configuration) (*application, error) {
	repo, err := repository.New(config.DSN())
	if err != nil {
		return nil, err
	}

	confWriter := confwriter.New(config.OutputFilepath, config.SkipDhcpNotification)
	ipAllocation := allocation.New(repo, config.CIDR)

	srv := service.New(repo, repo, repo, confWriter, ipAllocation)
	router := api.NewRouter(config, srv, srv, srv)

	return &application{
		repository: repo,
		service:    srv,
		router:     router,
	}, nil
}

func (app application) start() {
	log.Println("[INFO] Listening at :8080")
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           app.router,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	err := srv.ListenAndServe()
	panic(err)
}
