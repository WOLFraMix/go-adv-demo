package main

import (
	"fmt"
	"go-adv-demo/configs"
	"go-adv-demo/internal/auth"
	"go-adv-demo/internal/link"
	"go-adv-demo/pkg/db"
	"go-adv-demo/pkg/middleware"
	"net/http"
)

func main() {
	cfg := configs.LoadConfig()
	db := db.NewDb(cfg)
	router := http.NewServeMux()

	linkRepository := link.NewLinkRepository(db)

	auth.NewAuthHandler(router, auth.AuthHandlerDeps{
		Config: cfg,
	})
	link.NewLinkHandler(router, link.LinkHandlerDeps{
		LinkRepository: linkRepository,
	})

	server := http.Server{
		Addr:    ":8081",
		Handler: middleware.CORS(middleware.Logging(router)),
	}

	fmt.Println("Server is listening on port 8081...")
	server.ListenAndServe()
}
