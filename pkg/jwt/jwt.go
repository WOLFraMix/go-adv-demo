package jwt

import "github.com/golang-jwt/jwt/v5"

type JWT struct {
	Secret string
}

type JWTData struct {
	Email string
}

func NewJWT(secret string) *JWT {
	return &JWT{
		Secret: secret,
	}
}

func (j *JWT) Create(data JWTData) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": data.Email,
	})
	s, err := t.SignedString([]byte(j.Secret))
	if err != nil {
		return "", err
	}
	return s, nil
}

func (j *JWT) Parse(token string) (bool, *JWTData) {
	t, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		return []byte(j.Secret), nil
	})
	if err != nil {
		return false, nil
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return false, nil
	}
	email, ok := claims["email"].(string)
	if !ok {
		return false, nil
	}
	return t.Valid, &JWTData{
		Email: email,
	}
}
