package service

import (
	"encoding/base64"
	"encoding/json"

	"github.com/openpup/agora/internal/domain"
)

func encodeCursor(cursor domain.SignalListCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(payload), nil
}
