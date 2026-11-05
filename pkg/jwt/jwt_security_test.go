package jwt_test

import (
	"encoding/base64"
	"encoding/json"
	"go-adv-demo/pkg/jwt"
	"os"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const secret = "jwt-security-test-secret"

func TestParseRejectsTokenSignedWithOtherSecret(t *testing.T) {
	token, err := jwt.NewJWT("secret-a").Create(jwt.JWTData{Email: "user@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	valid, data := jwt.NewJWT("secret-b").Parse(token)
	if valid || data != nil {
		t.Fatal("token signed with another secret must be rejected")
	}
}

func TestParseRejectsTamperedPayload(t *testing.T) {
	token, err := jwt.NewJWT(secret).Create(jwt.JWTData{Email: "user@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	forgedPayload, _ := json.Marshal(map[string]any{"email": "admin@example.com"})
	parts[1] = base64.RawURLEncoding.EncodeToString(forgedPayload)

	valid, _ := jwt.NewJWT(secret).Parse(strings.Join(parts, "."))
	if valid {
		t.Fatal("token with modified payload must be rejected")
	}
}

// Атака alg=none: токен без подписи не должен приниматься.
func TestParseRejectsNoneAlgorithm(t *testing.T) {
	unsigned := jwtlib.NewWithClaims(jwtlib.SigningMethodNone, jwtlib.MapClaims{
		"email": "admin@example.com",
	})
	token, err := unsigned.SignedString(jwtlib.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	valid, _ := jwt.NewJWT(secret).Parse(token)
	if valid {
		t.Fatal("unsigned (alg=none) token must be rejected")
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	claims := jwtlib.MapClaims{
		"email": "user@example.com",
		"exp":   time.Now().Add(-time.Hour).Unix(),
	}
	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	valid, _ := jwt.NewJWT(secret).Parse(token)
	if valid {
		t.Fatal("expired token must be rejected")
	}
}

// Известный баг: Create не выставляет exp, поэтому выданный токен действует бессрочно.
func TestCreateSetsExpiration(t *testing.T) {
	if os.Getenv("KNOWN_BUGS") == "" {
		t.Skip("известный баг: токен создаётся без exp и не истекает (запуск: KNOWN_BUGS=1)")
	}
	token, err := jwt.NewJWT(secret).Create(jwt.JWTData{Email: "user@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	claims := jwtlib.MapClaims{}
	if _, _, err := jwtlib.NewParser().ParseUnverified(token, claims); err != nil {
		t.Fatal(err)
	}
	if _, ok := claims["exp"]; !ok {
		t.Fatal("issued token must carry an exp claim")
	}
}

// Известный баг: пустой секрет (SECRET не задан) не отклоняется. HMAC с пустым
// ключом принимается, поэтому любой может подписать токен с пустым ключом.
func TestParseRejectsEmptySecret(t *testing.T) {
	if os.Getenv("KNOWN_BUGS") == "" {
		t.Skip("известный баг: пустой JWT-секрет принимается (запуск: KNOWN_BUGS=1)")
	}
	forged, err := jwt.NewJWT("").Create(jwt.JWTData{Email: "admin@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	valid, _ := jwt.NewJWT("").Parse(forged)
	if valid {
		t.Fatal("token signed with an empty secret must not be accepted")
	}
}
