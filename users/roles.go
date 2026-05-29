package users

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

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
	ss := make([]string, 0, len(r))
	for _, role := range r {
		ss = append(ss, string(role))
	}
	b, _ := json.Marshal(ss)
	return string(b)
}

func (r *Roles) FromString(s string) error {
	var ss []string
	if err := json.Unmarshal([]byte(s), &ss); err != nil {
		return err
	}
	for _, s := range ss {
		*r = append(*r, Role(s))
	}
	return nil
}
