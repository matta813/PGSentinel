package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/matta813/pgsentinel/internal/models"
	"time"
)

func (s *Store) ListRuleProfiles(c context.Context) ([]models.RuleProfile, error) {
	r, e := s.DB.QueryContext(c, `SELECT id,name,description,entries_json,created_at,updated_at FROM rule_profiles ORDER BY name`)
	if e != nil {
		return nil, e
	}
	defer r.Close()
	o := []models.RuleProfile{}
	for r.Next() {
		var p models.RuleProfile
		var j, a, b string
		if e = r.Scan(&p.ID, &p.Name, &p.Description, &j, &a, &b); e != nil {
			return nil, e
		}
		_ = json.Unmarshal([]byte(j), &p.Entries)
		p.CreatedAt, _ = time.Parse(time.RFC3339Nano, a)
		p.UpdatedAt, _ = time.Parse(time.RFC3339Nano, b)
		o = append(o, p)
	}
	return o, r.Err()
}
func (s *Store) GetRuleProfile(c context.Context, id string) (models.RuleProfile, error) {
	var p models.RuleProfile
	var j, a, b string
	e := s.DB.QueryRowContext(c, `SELECT id,name,description,entries_json,created_at,updated_at FROM rule_profiles WHERE id=?`, id).Scan(&p.ID, &p.Name, &p.Description, &j, &a, &b)
	if e != nil {
		return p, e
	}
	_ = json.Unmarshal([]byte(j), &p.Entries)
	p.CreatedAt, _ = time.Parse(time.RFC3339Nano, a)
	p.UpdatedAt, _ = time.Parse(time.RFC3339Nano, b)
	return p, nil
}
func (s *Store) SaveRuleProfile(c context.Context, p *models.RuleProfile) error {
	j, e := json.Marshal(p.Entries)
	if e != nil {
		return e
	}
	n := time.Now().UTC()
	p.CreatedAt = n
	p.UpdatedAt = n
	_, e = s.DB.ExecContext(c, `INSERT INTO rule_profiles(id,name,description,entries_json,created_at,updated_at) VALUES(?,?,?,?,?,?)`, p.ID, p.Name, p.Description, string(j), stamp(n), stamp(n))
	return e
}
func (s *Store) DeleteRuleProfile(c context.Context, id string) error {
	r, e := s.DB.ExecContext(c, `DELETE FROM rule_profiles WHERE id=?`, id)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e == nil && n == 0 {
		return sql.ErrNoRows
	}
	return e
}
