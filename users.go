package app

import (
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
	var ss []string
	if err := json.Unmarshal([]byte(s), &ss); err != nil {
		return err
	}
	for _, s := range ss {
		*r = append(*r, Role(s))
	}
	return nil
}

type Session struct {
	ID      string   `json:"id"`
	UserID  uint     `json:"-"`
	IP      string   `json:"ip"`
	Agent   string   `json:"agent"`
	Time    DateTime `json:"time"`
	Expires DateTime `json:"expires"`
}

func (s Session) SQLInsert(tx *sql.Tx) (sql.Result, error) {
	return tx.Exec("INSERT INTO sessions (id, user_id, time, ip, agent, expires) VALUES (?, ?, ?, ?, ?, ?);",
		s.ID, s.UserID, s.Time, s.IP, s.Agent, s.Expires)
}
func (Session) SelectSQL(struct{}) string {
	return "SELECT id, ip, agent, time, expires FROM sessions WHERE user_id=? "
}
func (Session) CountSQL(struct{}) string   { return "SELECT COUNT(1) FROM sessions WHERE user_id=?;" }
func (Session) OrderSQL(struct{}) string   { return "ORDER BY time ASC " }
func (s Session) SQLParams(struct{}) []any { return []any{s.UserID} }
func (Session) Scan(rows *sql.Rows) (s Session, err error) {
	return s, rows.Scan(&s.ID, &s.IP, &s.Agent, &s.Time, &s.Expires)
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
	u.Salt = generateRandomID()
	u.Password = hashPassword(u.Salt, u.Password)
	return DBAdd(tx, u)
}

func (u User) SQLInsert(tx *sql.Tx) (sql.Result, error) {
	return tx.Exec("INSERT INTO users (email, salt, password, roles) VALUES (?, ?, ?, ?);",
		u.Email, u.Salt, u.Password, u.Roles)
}

func (u *User) ValidationTranslations() map[string]string {
	return map[string]string{
		"UNIQUE constraint failed: users.email": "Ja existeix un usuari amb aquest correu electrònic",
	}
}

func (User) SelectSQL(struct{}) string { return "SELECT id, email, salt, password, roles FROM users " }
func (User) CountSQL(struct{}) string  { return "SELECT COUNT(1) FROM users;" }
func (User) OrderSQL(struct{}) string  { return "ORDER BY email ASC " }
func (User) SQLParams(struct{}) []any  { return nil }
func (User) Scan(rows *sql.Rows) (u User, err error) {
	return u, rows.Scan(&u.ID, &u.Email, &u.Salt, &u.Password, &u.Roles)
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
