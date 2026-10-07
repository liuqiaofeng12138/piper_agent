package meta

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", filepath.ToSlash(path))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{DB: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

func (s *Store) migrate() error {
	tables := []string{TableIndices, TableFunctions, TableVarsLists, TableAccounts, TableProxies, TableTemplates, TableTasks, TableNodes}
	for _, t := range tables {
		stmts := []string{
			fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
  id TEXT PRIMARY KEY,
  name TEXT,
  username TEXT,
  domain TEXT,
  group_ TEXT,
  ssh_host TEXT,
  status TEXT,
  tpl_id TEXT,
  create_time INTEGER NOT NULL,
  update_time INTEGER NOT NULL,
  payload TEXT NOT NULL
)`, t),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_name ON %s(name)`, t, t),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_domain ON %s(domain)`, t, t),
		}
		for _, stmt := range stmts {
			if _, err := s.DB.Exec(stmt); err != nil {
				return err
			}
		}
	}
	return nil
}
