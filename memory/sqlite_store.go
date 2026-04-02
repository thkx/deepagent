//go:build sqlite
// +build sqlite

package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
)

type sqliteStore struct {
	db *sql.DB
}

func newSQLiteStore(dsn string) (Store, error) {
	if dsn == "" {
		dsn = "memory.db"
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	// create table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS memory (namespace TEXT, key TEXT, value BLOB, PRIMARY KEY(namespace,key))`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &sqliteStore{db: db}, nil
}

func (s *sqliteStore) Get(ctx context.Context, namespace, key string) (any, bool, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT value FROM memory WHERE namespace = ? AND key = ?`, namespace, key).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, false, err
	}
	return v, true, nil
}

func (s *sqliteStore) Put(ctx context.Context, namespace, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO memory(namespace,key,value) VALUES(?,?,?) ON CONFLICT(namespace,key) DO UPDATE SET value=excluded.value`, namespace, key, data)
	return err
}

func (s *sqliteStore) List(ctx context.Context, namespace string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key FROM memory WHERE namespace = ?`, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *sqliteStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
