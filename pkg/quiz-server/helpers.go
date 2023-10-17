package quiz_server

import "github.com/google/uuid"

func generateUniqueID() string {
	return uuid.New().String()
}
