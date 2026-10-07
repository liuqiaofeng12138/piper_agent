package meta

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNotFound = errors.New("not found")

type QueryOpts struct {
	Q      string
	Page   int64
	Size   int64
	Status string
	Domain string
}

func (s *Store) Query(table string, opt QueryOpts) (items []map[string]any, total int64, err error) {
	if opt.Page < 1 {
		opt.Page = 1
	}
	if opt.Size < 1 {
		opt.Size = 10
	}
	q := opt.Q + "%"
	where := []string{"(id LIKE ? OR IFNULL(name,'') LIKE ?)"}
	args := []any{q, q}

	if table == TableAccounts {
		where[0] = "(id LIKE ? OR IFNULL(username,'') LIKE ? OR IFNULL(domain,'') LIKE ?)"
		args = []any{q, q, q}
	}
	if table == TableProxies {
		where[0] = "(id LIKE ? OR IFNULL(group_,'') LIKE ? OR IFNULL(ssh_host,'') LIKE ?)"
		args = []any{q, q, q}
		if opt.Status != "" {
			where = append(where, "status = ?")
			args = append(args, opt.Status)
		}
	}

	whereSQL := strings.Join(where, " AND ")
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, whereSQL)
	if err = s.DB.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listSQL := fmt.Sprintf(`SELECT payload FROM %s WHERE %s ORDER BY update_time DESC LIMIT ? OFFSET ?`, table, whereSQL)
	args = append(args, opt.Size, (opt.Page-1)*opt.Size)
	rows, err := s.DB.Query(listSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var payload string
		if err = rows.Scan(&payload); err != nil {
			return nil, 0, err
		}
		var m map[string]any
		if err = json.Unmarshal([]byte(payload), &m); err != nil {
			return nil, 0, err
		}
		items = append(items, m)
	}
	return items, total, rows.Err()
}

func (s *Store) Get(table, id string) (map[string]any, error) {
	var payload string
	err := s.DB.QueryRow(fmt.Sprintf("SELECT payload FROM %s WHERE id = ?", table), id).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) GetByField(table, field, value string) (map[string]any, error) {
	q := fmt.Sprintf("SELECT payload FROM %s WHERE %s = ? LIMIT 1", table, field)
	var payload string
	err := s.DB.QueryRow(q, value).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) Upsert(table string, doc map[string]any) error {
	id, _ := doc["id"].(string)
	if id == "" {
		return errors.New("id required")
	}
	now := time.Now().UnixMilli()
	if _, ok := doc["create_time"]; !ok {
		doc["create_time"] = now
	}
	doc["update_time"] = now

	payload, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	name := strField(doc, "name")
	username := strField(doc, "username")
	domain := strField(doc, "domain")
	group := strField(doc, "group_")
	if group == "" {
		group = strField(doc, "group")
	}
	sshHost := strField(doc, "ssh_host")
	status := strField(doc, "status")
	tplID := strField(doc, "tpl_id")

	_, err = s.DB.Exec(fmt.Sprintf(`
INSERT INTO %s (id,name,username,domain,group_,ssh_host,status,tpl_id,create_time,update_time,payload)
VALUES (?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  name=excluded.name, username=excluded.username, domain=excluded.domain,
  group_=excluded.group_, ssh_host=excluded.ssh_host, status=excluded.status,
  tpl_id=excluded.tpl_id, update_time=excluded.update_time, payload=excluded.payload
`, table), id, name, username, domain, group, sshHost, status, tplID, doc["create_time"], doc["update_time"], string(payload))
	return err
}

func (s *Store) Delete(table, id string) error {
	res, err := s.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", table), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Distinct(table, column, likePrefix string) ([]string, error) {
	q := fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE %s LIKE ? AND %s IS NOT NULL AND %s != ''", column, table, column, column, column)
	rows, err := s.DB.Query(q, likePrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) DistinctEq(table, column, eq string) ([]string, error) {
	q := fmt.Sprintf("SELECT DISTINCT username FROM %s WHERE domain = ?", table)
	rows, err := s.DB.Query(q, eq)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func strField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func ParseJSONObject(body []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func ParseJSONArray(body []byte) ([]map[string]any, error) {
	var arr []map[string]any
	if err := json.Unmarshal(body, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}
