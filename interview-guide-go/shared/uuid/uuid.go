package uuid

import "github.com/google/uuid"

func NewUUID16() string {
	return uuid.New().String()[:16]
}
