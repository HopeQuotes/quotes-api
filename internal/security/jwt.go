package security

import (
	"github.com/google/uuid"
	"github.com/pascaldekloe/jwt"
	"time"
)

func NewJWT(userID uuid.UUID, expiry time.Time, issuer string, secreteKey string) (string, error) {

	var claims jwt.Claims
	claims.Subject = userID.String()

	claims.Issued = jwt.NewNumericTime(time.Now())
	claims.NotBefore = jwt.NewNumericTime(time.Now())
	claims.Expires = jwt.NewNumericTime(expiry)

	claims.Issuer = issuer
	claims.Audiences = []string{issuer}
	claims.Audiences = []string{issuer}

	jwtBytes, err := claims.HMACSign(jwt.HS256, []byte(secreteKey))
	if err != nil {
		return "", err
	}

	return string(jwtBytes), nil
}
