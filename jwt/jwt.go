package jwt

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
	"github.com/ipfans/components/v2/utils"
)

type Manager struct {
	sig jose.Signer
	cfg *Config
}

type Config struct {
	SecretKey string        `koanf:"SecretKey"` // Secret key for signing the token, required. The key should match the algorithm.
	Algorithm string        `koanf:"Algorithm"` // Algorithm of the token, default is HS256. Supported algorithms: HS256, HS384, HS512
	Expire    time.Duration `koanf:"Expire"`    // Expire time of the token, default is 24 hours
}

func cutOrPaddingKey(key string, length int) string {
	size := len([]byte(key))
	if size < length {
		return key + strings.Repeat("0", length-size)
	}
	return key[:length]
}

func New(cfg *Config) *Manager {
	cfg.Expire = utils.DefaultValue(cfg.Expire, time.Hour*24)
	cfg.Algorithm = utils.DefaultValue(cfg.Algorithm, "HS256")

	var algorithm jose.SignatureAlgorithm
	switch cfg.Algorithm {
	case "HS256":
		algorithm = jose.HS256
		cfg.SecretKey = cutOrPaddingKey(cfg.SecretKey, 32)
	case "HS384":
		algorithm = jose.HS384
		cfg.SecretKey = cutOrPaddingKey(cfg.SecretKey, 48)
	case "HS512":
		algorithm = jose.HS512
		cfg.SecretKey = cutOrPaddingKey(cfg.SecretKey, 64)
	default:
		panic("unsupported algorithm: " + cfg.Algorithm)
	}

	sig, err := jose.NewSigner(jose.SigningKey{Algorithm: algorithm, Key: []byte(cfg.SecretKey)}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		panic(err)
	}

	return &Manager{
		sig: sig,
		cfg: cfg,
	}
}

// Generate generates a JWT token for the given user ID. It returns the token and the id of the token.
func (m *Manager) Generate(userID uint) (string, string, error) {
	id := uuid.New().String()
	now := time.Now()
	claims := jwt.Claims{
		Subject:  fmt.Sprintf("%d", userID),
		IssuedAt: jwt.NewNumericDate(now),
		Expiry:   jwt.NewNumericDate(now.Add(m.cfg.Expire)),
		ID:       id,
	}

	token, err := jwt.Signed(m.sig).Claims(claims).Serialize()
	return token, id, err
}

// Parser parses the JWT token and returns the claims.
func (m *Manager) Parser(token string) (jwt.Claims, error) {
	var claims jwt.Claims
	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.SignatureAlgorithm(m.cfg.Algorithm)})
	if err != nil {
		return claims, err
	}

	err = parsed.Claims([]byte(m.cfg.SecretKey), &claims)
	return claims, err
}
