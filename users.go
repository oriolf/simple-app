package app

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type User struct {
	ID       uint      `json:"id"`
	Email    string    `json:"email"`
	Salt     string    `json:"-"`
	Password string    `json:"-"`
	Roles    Roles     `json:"roles"`
	Sessions []Session `json:"sessions"`
}

type Role string

type Roles []Role

const (
	RoleSuperUser = Role("superuser")
)

func (r Roles) MarshalJSON() ([]byte, error) {
	return []byte(r.String()), nil
}

func (r *Roles) UnmarshalJSON(b []byte) error {
	if err := r.FromString(string(b)); err != nil {
		return fmt.Errorf("could not scan roles: %w", err)
	}

	return nil
}

func (r Roles) Value() (driver.Value, error) {
	return r.String(), nil
}

func (r *Roles) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("must receive a non-null string")
	}
	switch value := value.(type) {
	case string:
		return r.FromString(value)
	case []byte:
		return r.FromString(string(value))
	}

	return fmt.Errorf("must receive a string")
}

func (r Roles) String() string {
	b, _ := json.Marshal(r)
	return string(b)
}

func (r *Roles) FromString(s string) error {
	return json.Unmarshal([]byte(s), r)
}

type Session struct {
	ID      string   `json:"id"`
	IP      string   `json:"ip"`
	Agent   string   `json:"agent"`
	Expires DateTime `json:"expires"`
}
