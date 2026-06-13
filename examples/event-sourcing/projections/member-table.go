package projections

import (
	"database/sql"

	"github.com/oriolf/simple-app/db"
	"github.com/oriolf/simple-app/examples/event-sourcing/entities"
	"github.com/oriolf/simple-app/types"

	app "github.com/oriolf/simple-app"
	de "github.com/oriolf/simple-app/domain-events"
	vo "github.com/oriolf/simple-app/examples/event-sourcing/value-objects"
)

// response for api, contract must hold or else it breaks clients
// also have a table associated which is directly mapped
type MemberTable struct {
	ID       types.UUID  `json:"id"`
	Name     vo.Name     `json:"name"`
	NIF      vo.NIF      `json:"nif"`
	JoinedOn types.Date  `json:"joined_on"`
	LeftOn   *types.Date `json:"left_on"`
}

type memberTableProjecter struct{}

var MemberTableProjecter memberTableProjecter

func (p memberTableProjecter) TableName() string { return "members_table" }

func (p memberTableProjecter) Project(tx *sql.Tx, events []de.DomainEvent) error {
	m := &entities.Member{}
	de.Hydrate(m, events)

	// TODO if projection already exists, ON CONFLICT UPDATE ...
	sql := "INSERT INTO members_table (id, projected_on, nif, name, joined_on, left_on) VALUES (?, ?, ?, ?, ?, ?);"
	_, err := tx.Exec(sql, m.ID, types.Now(), m.NIF, m.Name, m.JoinedOn, m.LeftOn)
	return err
}

func (m MemberTable) List(p app.Paginator, c memberFilterCriteria) ([]MemberTable, uint, error) {
	return db.List(m, p, c)
}

// HTTP methods
type memberFilterCriteria struct {
	search string
}

func (m MemberTable) FilterCriteria(params map[string]any) memberFilterCriteria {
	return memberFilterCriteria{search: app.SafeGetString(params, "search")}
}

func (m MemberTable) SelectSQL(criteria memberFilterCriteria) string {
	return "SELECT id, name, nif, joined_on, left_on FROM members_table " + m.whereSQL(criteria)
}

func (MemberTable) OrderSQL(criteria memberFilterCriteria) string { return "ORDER BY joined_on DESC " }

func (MemberTable) Scan(rows *sql.Rows) (m MemberTable, err error) {
	return m, rows.Scan(&m.ID, &m.Name, &m.NIF, &m.JoinedOn, &m.LeftOn)
}

func (m MemberTable) CountSQL(criteria memberFilterCriteria) string {
	return "SELECT COUNT(1) FROM members_table" + m.whereSQL(criteria) + ";"
}

func (MemberTable) whereSQL(criteria memberFilterCriteria) string {
	if criteria.search != "" {
		return " WHERE (name LIKE @search OR nif LIKE @search)"
	}
	return ""
}

func (MemberTable) SQLWhereCriteriaParams(criteria memberFilterCriteria) []any {
	if criteria.search != "" {
		search := "%" + criteria.search + "%"
		return []any{sql.Named("search", search)}
	}
	return nil
}
