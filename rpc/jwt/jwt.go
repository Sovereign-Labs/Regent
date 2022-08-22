package jwt

import (
	"errors"
	"fmt"
	"io/ioutil"
	"regent/common"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
)

type EthJwt struct {
	issuedAt     time.Time
	signedString string
	secret       []byte
}

// Refreshes the jwt. This is done automatically, so the method is private
func (ethJwt *EthJwt) refresh() error {
	ethJwt.issuedAt = time.Now()

	// Per the ethereum spec, valid JWTs have two claims - issued at (iat), and client version (clv)
	// The token must use HMAC-SHA256.
	// https://github.com/ethereum/execution-apis/blob/main/src/engine/authentication.md
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"clv": common.VERSION_STRING,
		"iat": ethJwt.issuedAt.Unix(),
	})
	signedString, err := token.SignedString(ethJwt.secret)
	ethJwt.signedString = signedString
	if err != nil {
		return fmt.Errorf("the jwt expired but could not be refreshed. err: %w", err)
	}
	return nil
}

// Returns the signed token string for the jwt, refreshing the token if necessary
func (token *EthJwt) TokenString() (string, error) {
	if time.Since(token.issuedAt) > time.Second*55 {
		err := token.refresh()
		if err != nil {
			return "", err
		}
	}
	return token.signedString, nil
}

func FromSecret(secret []byte) *EthJwt {
	return &EthJwt{
		secret: secret,
	}
}

func FromSecretFile(filename string) (*EthJwt, error) {
	rawSecret, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	jwtSecret := common.FromHex(strings.TrimSpace(string(rawSecret)))
	if len(jwtSecret) != 32 {
		return nil, errors.New("invalid JWT secret")
	}
	return &EthJwt{
		secret: jwtSecret,
	}, nil
}
