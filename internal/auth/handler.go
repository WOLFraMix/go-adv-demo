package auth

import (
	"fmt"
	"go-adv-demo/configs"
	"go-adv-demo/pkg/req"
	"go-adv-demo/pkg/resp"
	"net/http"
)

type AuthHandler struct {
	*configs.Config
	*AuthService
}

type AuthHandlerDeps struct {
	*configs.Config
	*AuthService
}

func NewAuthHandler(router *http.ServeMux, deps AuthHandlerDeps) {
	handler := &AuthHandler{
		Config:      deps.Config,
		AuthService: deps.AuthService,
	}
	router.HandleFunc("POST /auth/login", handler.Login())
	router.HandleFunc("POST /auth/register", handler.Register())
}

func (handler *AuthHandler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[LoginRequest](&w, r)
		if err != nil {
			return
		}

		fmt.Println("Login")
		fmt.Println(body)
		data := LoginResponse{
			Token: "321",
		}
		resp.Json(w, data, 200)
	}
}

func (handler *AuthHandler) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[RegistRequest](&w, r)
		if err != nil {
			return
		}

		fmt.Println("Register")
		handler.AuthService.Register(body.Email, body.Password, body.Name)
		data := LoginResponse{
			Token: "DONE",
		}
		resp.Json(w, data, 201)
	}
}
