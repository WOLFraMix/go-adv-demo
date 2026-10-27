package main

import (
	"fmt"
	"go-adv-demo/configs"
	"go-adv-demo/internal/auth"
	"go-adv-demo/internal/link"
	"go-adv-demo/internal/user"
	"go-adv-demo/pkg/db"
	"go-adv-demo/pkg/event"
	"go-adv-demo/pkg/middleware"
	"net/http"
)

func main() {
	cfg := configs.LoadConfig()
	db := db.NewDb(cfg)
	router := http.NewServeMux()
	eventBus := event.NewEventBus()

	linkRepository := link.NewLinkRepository(db)
	userRepository := user.NewUserRepository(db)
	// statRepository := stat.NewStatRepository(db)

	authServices := auth.NewAuthService(userRepository)

	auth.NewAuthHandler(router, auth.AuthHandlerDeps{
		Config:      cfg,
		AuthService: authServices,
	})
	link.NewLinkHandler(router, link.LinkHandlerDeps{
		LinkRepository: linkRepository,
		EventBus:       eventBus,
		Config:         cfg,
	})

	stack := middleware.Chain(
		middleware.CORS,
		middleware.Logging,
	)

	server := http.Server{
		Addr:    ":8081",
		Handler: stack(router),
	}

	fmt.Println("Server is listening on port 8081...")
	server.ListenAndServe()
}
