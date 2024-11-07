package password

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/ipfans/components/v2/utils"
	"golang.org/x/crypto/argon2"
)

type Config struct {
	Time    uint32 // 迭代次数, default: 1
	Memory  uint32 // 内存大小, default: 64 * 1024
	Threads uint8  // 线程数, default: 4
	KeyLen  uint32 // 密钥长度, default: 32
}

type Manager struct {
	cfg *Config
}

// New create a password manager
func New(configs ...*Config) *Manager {
	var cfg *Config
	if len(configs) > 0 {
		cfg = configs[0]
	} else {
		cfg = &Config{
			Time:    1,
			Memory:  64 * 1024,
			Threads: 4,
			KeyLen:  32,
		}
	}

	cfg.Time = utils.DefaultValue(cfg.Time, 1)
	cfg.Memory = utils.DefaultValue(cfg.Memory, 64*1024)
	cfg.Threads = utils.DefaultValue(cfg.Threads, 4)
	cfg.KeyLen = utils.DefaultValue(cfg.KeyLen, 32)

	return &Manager{
		cfg: cfg,
	}
}

func (m *Manager) Generate(password string) string {
	salt := make([]byte, m.cfg.KeyLen)
	_, _ = rand.Read(salt)
	hash := argon2.IDKey([]byte(password), salt, m.cfg.Time, m.cfg.Memory, m.cfg.Threads, m.cfg.KeyLen)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		m.cfg.Memory,
		m.cfg.Time,
		m.cfg.Threads,
		b64Salt,
		b64Hash)
}

func (m *Manager) Verify(password, hash string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false
	}
	if parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return false
	}
	paramsMap := make(map[string]uint32)
	for _, param := range params {
		kv := strings.Split(param, "=")
		if len(kv) != 2 {
			return false
		}
		m, err := strconv.ParseUint(kv[1], 10, 32)
		if err != nil {
			return false
		}
		paramsMap[kv[0]] = uint32(m)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	return bytes.Equal(argon2.IDKey([]byte(password), salt, paramsMap["t"], paramsMap["m"], uint8(paramsMap["p"]), uint32(len(decodedHash))), decodedHash)
}
