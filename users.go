package app

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
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
	ss := make([]string, 0, len(r))
	for _, role := range r {
		ss = append(ss, string(role))
	}
	b, _ := json.Marshal(ss)
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

var CLIAddSuperUser = CLIAdd(SuperUserFactory)

func SuperUserFactory() *User { return &User{Roles: Roles{RoleSuperUser}} }

func (u *User) Validate(params map[string]any) ApiErrors {
	v := NewValidator(params)
	u.Email = v.ValidateEmail("email")
	u.Password = v.ValidatePassword("password")
	return v.Errors()
}

func (u User) Add(tx *sql.Tx) (uint, error) {
	u.Salt = generateSalt()
	u.Password = hashPassword(u.Salt, u.Password)
	return DBAdd(tx, u)
}

func (u User) SQLInsert(tx *sql.Tx) (sql.Result, error) {
	return tx.Exec("INSERT INTO users (email, salt, password, roles) VALUES (?, ?, ?, ?);",
		u.Email, u.Salt, u.Password, u.Roles)
}

func (u User) ValidationTranslations() map[string]string {
	return map[string]string{
		"UNIQUE constraint failed: users.email": "Ja existeix un usuari amb aquest correu electrònic",
	}
}

func hashPassword(salt, password string) string {
	hash := salt + password
	for i := 0; i < 1000; i++ {
		hasher := sha256.New()
		hasher.Write([]byte(hash))
		hash = hex.EncodeToString(hasher.Sum(nil))
	}
	return hash
}

func generateSalt() string {
	salt := make([]byte, 32)
	rand.Read(salt)
	return hex.EncodeToString(salt)
}
