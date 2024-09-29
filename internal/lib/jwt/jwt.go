package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/BariVakhidov/sso/internal/domain/models"
)

// NewToken generates new JWT token and returns tokenString and err
func NewToken(user *models.User, app models.App, duration time.Duration) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["email"] = user.Email
	claims["uid"] = user.ID
	claims["exp"] = time.Now().Add(duration).Unix()
	claims["app_id"] = app.ID

	tokenString, err := token.SignedString([]byte(app.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
