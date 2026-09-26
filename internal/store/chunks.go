package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"regexp"
	"unicode/utf8"
)

var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (s *Store) SaveChunk(ctx context.Context, p c.Principal, hash string, index, total int, body []byte) error {
	if p.Device == "" {
		return c.ErrForbidden
	}
	if !digestPattern.MatchString(hash) || total < 1 || total > 512 || index < 0 || index >= total || len(body) == 0 || len(body) > 1<<20 {
		return c.ErrInvalid
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", p.Workspace+":"+p.Device); e != nil {
		return e
	}
	var existing []byte
	var count, oldTotal int
	var size int64
	if e = tx.QueryRow(ctx, "SELECT count(*),coalesce(sum(octet_length(body)),0) FROM tm_chunks WHERE workspace=$1 AND device=$2", p.Workspace, p.Device).Scan(&count, &size); e != nil {
		return e
	}
	if size+int64(len(body)) > 1<<30 {
		return c.ErrInvalid
	}
	if _, e = tx.Exec(ctx, `INSERT INTO tm_chunks(workspace,device,hash,part,total,body) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, p.Workspace, p.Device, hash, index, total, body); e != nil {
		return e
	}
	if e = tx.QueryRow(ctx, "SELECT total,body FROM tm_chunks WHERE workspace=$1 AND device=$2 AND hash=$3 AND part=$4", p.Workspace, p.Device, hash, index).Scan(&oldTotal, &existing); e != nil {
		return e
	}
	if oldTotal != total || !bytes.Equal(existing, body) {
		return c.ErrConflict
	}
	return tx.Commit(ctx)
}
func (s *Store) CommitChunks(ctx context.Context, p c.Principal, hash string) (c.Session, error) {
	var empty c.Session
	if p.Device == "" {
		return empty, c.ErrForbidden
	}
	if !digestPattern.MatchString(hash) {
		return empty, c.ErrInvalid
	}
	rows, e := s.DB.Query(ctx, "SELECT part,total,body FROM tm_chunks WHERE workspace=$1 AND device=$2 AND hash=$3 ORDER BY part", p.Workspace, p.Device, hash)
	if e != nil {
		return empty, e
	}
	var body []byte
	index, total := 0, 0
	for rows.Next() {
		var i, n int
		var part []byte
		if e = rows.Scan(&i, &n, &part); e != nil {
			rows.Close()
			return empty, e
		}
		if i != index || (index > 0 && total != n) {
			rows.Close()
			return empty, c.ErrConflict
		}
		index++
		total = n
		body = append(body, part...)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return empty, e
	}
	if index != total || total == 0 || Hash(body) != hash {
		return empty, c.ErrConflict
	}
	var v c.Snapshot
	if !utf8.Valid(body) || json.Unmarshal(body, &v) != nil {
		return empty, c.ErrInvalid
	}
	row, e := s.Accept(ctx, p, v)
	if e != nil && !errors.Is(e, c.ErrDeleted) {
		return empty, e
	}
	terminal := e
	_, e = s.DB.Exec(ctx, "DELETE FROM tm_chunks WHERE workspace=$1 AND device=$2 AND hash=$3", p.Workspace, p.Device, hash)
	if e != nil {
		return row, e
	}
	return row, terminal
}
