package users

import (
	"database/sql"

	"github.com/oriolf/simple-app/types"
)

type Session struct {
	ID      string         `json:"id"`
	UserID  uint           `json:"-"`
	IP      string         `json:"ip"`
	Agent   string         `json:"agent"`
	Time    types.DateTime `json:"time"`
	Expires types.DateTime `json:"expires"`
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
