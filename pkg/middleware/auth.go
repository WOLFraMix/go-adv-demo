package middleware

import (
	"fmt"
	"go-adv-demo/configs"
	"go-adv-demo/pkg/jwt"
	"net/http"
	"strings"
)

func IsAuth(next http.Handler, config *configs.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		isValid, data := jwt.NewJWT(config.Auth.Secret).Parse(token)
		fmt.Println(isValid)
		fmt.Println(data)
		fmt.Println(token)
		next.ServeHTTP(w, r)
	})
}
