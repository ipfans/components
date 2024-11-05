package utils

import (
	"crypto/rand"

	"github.com/google/uuid"
)

func NewUUID() string {
	return uuid.Must(uuid.NewRandomFromReader(rand.Reader)).String()
}
