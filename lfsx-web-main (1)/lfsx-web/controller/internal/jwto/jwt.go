package jwto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Username   string `json:"sub"`
	Database   string `json:"h_d"`
	DbPassword string `json:"h_p"`
	DbUser     string `json:"h_u"`
	Workplace  string `json:"h_ap"`
	Expiration int    `json:"exp"`
	jwt.RegisteredClaims
}

// ValidateToken validates the given token. Authroized returns if the token and
// the expiry date were still valid
func ValidateToken(token string, key []byte) (claim *Claims, authorized bool, err error) {
	claims := &Claims{}
	tkn, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})

	if err != nil || !tkn.Valid {
		return nil, false, err
	}

	return claims, true, nil
}

// ToUser converts the encrypted claim fields
// to a user struct.
// For decrypting the fields the key for ASE Encryption
// is needed.
func (c *Claims) ToUser(key []byte) (*models.User, error) {
	// Hash the key with SHA-256
	hasher := sha256.New()
	hasher.Write(key)
	hash := hasher.Sum(nil)

	// Create ASE cipher with only the first 16 Byte
	cipher, err := aes.NewCipher(hash[0:16])
	if err != nil {
		return nil, err
	}

	// Decrypt
	rtc := &models.User{
		Username:    c.Username,
		DbPassword:  decrypt(cipher, c.DbPassword),
		DbUser:      decrypt(cipher, c.DbUser),
		DatabaseStr: decrypt(cipher, c.Database),
		Workplace:   decrypt(cipher, c.Workplace),
		Expiration:  c.Expiration,
		Database:    models.NewDatabase(decrypt(cipher, c.Database)),
	}

	return rtc, nil
}

func decrypt(c cipher.Block, val string) string {
	// The value is base64 decoded
	v, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		logger.Warning("Base64 decoding failed: %s", err)
		return ""
	}

	nonceSize := 12
	// Check if the IV is really contained in the encrypted value
	if len(v) < nonceSize {
		logger.Warning("Received value to decrypt with a length that is smaller as the IV size: %d - %d", len(v), nonceSize)
		return ""
	}

	// Read out the GCM cipher from the encrypted value
	gcm, err := cipher.NewGCMWithNonceSize(c, nonceSize)
	if err != nil {
		logger.Warning("Creation of GCM failed: %s", err)
		return ""
	}

	// Decrypt the value
	plaintext, err := gcm.Open(nil, v[:nonceSize], v[nonceSize:], nil)
	if err != nil {
		logger.Warning("Failed to decrypt AES value: %s", err)
		return ""
	}
	return string(plaintext)
}
