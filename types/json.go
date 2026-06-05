package types

import (
	"encoding/json"
)

// TODO payload is json, methods Payload() (map[string]any, error) and TypedPayload(target any) error

type JSON json.RawMessage
