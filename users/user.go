package users

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"

	app "github.com/oriolf/simple-app"
	"github.com/oriolf/simple-app/db"
	"github.com/oriolf/simple-app/validators"
)

type User struct {
	ID       uint      `json:"id"`
	Email    string    `json:"email"`
	Salt     string    `json:"-"`
	Password string    `json:"-"`
	Roles    Roles     `json:"roles"`
	Sessions []Session `json:"sessions"`
}

func SuperUserFactory() *User { return &User{Roles: Roles{RoleSuperUser}} }

func (u *User) Validate(params map[string]any) app.ApiErrors {
	v := validators.NewValidator(params)
	u.Email = v.ValidateEmail("email")
	u.Password = v.ValidatePassword("password")
	return v.Errors()
}

func (u User) CorrectPassword(password string) bool {
	passwordHash := hashPassword(u.Salt, password)
	return passwordHash == u.Password
}

func (u User) Add(tx *sql.Tx) (uint, error) {
	u.Salt = app.GenerateRandomID()
	u.Password = hashPassword(u.Salt, u.Password)
	return db.Add(tx, u)
}

func (u User) SQLInsert(tx *sql.Tx) (sql.Result, error) {
	return tx.Exec("INSERT INTO users (email, salt, password, roles) VALUES (?, ?, ?, ?);",
		u.Email, u.Salt, u.Password, u.Roles)
}

func (User) SelectSQL(struct{}) string { return "SELECT id, email, salt, password, roles FROM users " }
func (User) OrderSQL(struct{}) string  { return "ORDER BY email ASC " }

func (User) Scan(rows *sql.Rows) (u User, err error) {
	return u, rows.Scan(&u.ID, &u.Email, &u.Salt, &u.Password, &u.Roles)
}

func (User) CountSQL(struct{}) string { return "SELECT COUNT(1) FROM users;" }

func hashPassword(salt, password string) string {
	hash := salt + password
	for i := 0; i < 1000; i++ {
		hasher := sha256.New()
		hasher.Write([]byte(hash))
		hash = hex.EncodeToString(hasher.Sum(nil))
	}
	return hash
}
