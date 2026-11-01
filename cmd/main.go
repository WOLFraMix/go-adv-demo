package main

import (
	"fmt"
	"go-adv-demo/configs"
	"go-adv-demo/internal/auth"
	"go-adv-demo/internal/link"
	"go-adv-demo/internal/stat"
	"go-adv-demo/internal/user"
	"go-adv-demo/pkg/db"
	"go-adv-demo/pkg/event"
	"go-adv-demo/pkg/middleware"
	"log"
	"net/http"
)

func App() http.Handler {
	cfg := configs.LoadConfig()
	db := db.NewDb(cfg)
	router := http.NewServeMux()
	eventBus := event.NewEventBus()

	linkRepository := link.NewLinkRepository(db)
	userRepository := user.NewUserRepository(db)
	statRepository := stat.NewStatRepository(db)

	authServices := auth.NewAuthService(userRepository)
	statService := stat.NewStatService(&stat.StatServiceDeps{
		EventBus:       eventBus,
		StatRepository: statRepository,
	})

	auth.NewAuthHandler(router, auth.AuthHandlerDeps{
		Config:      cfg,
		AuthService: authServices,
	})
	link.NewLinkHandler(router, link.LinkHandlerDeps{
		LinkRepository: linkRepository,
		EventBus:       eventBus,
		Config:         cfg,
	})
	stat.NewStatHandler(router, stat.StatHandlerDeps{
		StatRepository: statRepository,
		Config:         cfg,
	})

	go statService.AddClick()

	stack := middleware.Chain(
		middleware.CORS,
		middleware.Logging,
	)

	return stack(router)
}

func main() {
	app := App()
	server := http.Server{
		Addr:    ":8081",
		Handler: app,
	}

	fmt.Println("Server is listening on port 8081...")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
