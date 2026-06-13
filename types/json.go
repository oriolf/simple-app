package types

import (
	"encoding/json"
	"fmt"
)

type JSON json.RawMessage

func NewJSON(m map[string]any) JSON {
	b, err := json.Marshal(m)
	if err != nil {
		panic(fmt.Errorf("could not marshal json: %s", err))
	}
	return JSON(b)
}

func (j JSON) GetMap() (m map[string]any) {
	if err := json.Unmarshal(j, &m); err != nil {
		panic(fmt.Errorf("could not unmarshal json: %s", err))
	}
	return m
}
