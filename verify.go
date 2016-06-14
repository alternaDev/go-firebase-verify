package firebase

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

const (
	clientCertURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"
)

func VerifyIDToken(idToken string, googleProjectID string) (string, error) {
	keys, err := fetchPublicKeys()

	if err != nil {
		return "", err
	}

	parsedToken, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		kid := token.Header["kid"]

		certPEM := string(*keys[kid.(string)])
		certPEM = strings.Replace(certPEM, "\\n", "\n", -1)
		certPEM = strings.Replace(certPEM, "\"", "", -1)
		block, _ := pem.Decode([]byte(certPEM))
		var cert *x509.Certificate
		cert, _ = x509.ParseCertificate(block.Bytes)
		rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)

		return rsaPublicKey, nil
	})

	if err != nil {
		return "", err
	}

	errMessage := ""

	if parsedToken.Claims["aud"].(string) != googleProjectID {
		errMessage = "Firebase Auth ID token has incorrect 'aud' claim: " + parsedToken.Claims["aud"].(string)
	} else if parsedToken.Claims["iss"].(string) != "https://securetoken.google.com/"+googleProjectID {
		errMessage = "Firebase Auth ID token has incorrect 'iss' claim"
	} else if parsedToken.Claims["sub"].(string) == "" || len(parsedToken.Claims["sub"].(string)) > 128 {
		errMessage = "Firebase Auth ID token has invalid 'sub' claim"
	}

	if errMessage != "" {
		return "", errors.New(errMessage)
	}

	return string(parsedToken.Claims["sub"].(string)), nil
}

func fetchPublicKeys() (map[string]*json.RawMessage, error) {
	resp, err := http.Get(clientCertURL)

	if err != nil {
		return nil, err
	}

	var objmap map[string]*json.RawMessage
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&objmap)

	return objmap, err
}
