// Package store owns PostgreSQL persistence and permissioned archive reads.
package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/policy"
	"strings"
	"time"

	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	DB        *pgxpool.Pool
	ViewerKey []byte
}

func Open(ctx context.Context, url string) (*Store, error) {
	if url == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	key, err := viewerEnvKey()
	if err != nil {
		db.Close()
		return nil, err
	}
	s := &Store{DB: db, ViewerKey: key}
	if err = s.Migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Migrate(ctx context.Context) error {
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(76400219)"); err != nil {
		return err
	}
	for _, file := range files {
		b, e := migrations.ReadFile("migrations/" + file.Name())
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, string(b)); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
func ID() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func Hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func (s *Store) Policy(ctx context.Context, workspace string) (c.Policy, error) {
	var b []byte
	var p c.Policy
	err := s.DB.QueryRow(ctx, "SELECT body FROM tm_policies WHERE workspace=$1", workspace).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, c.ErrNotFound
	}
	if err != nil {
		return p, err
	}
	err = json.Unmarshal(b, &p)
	return p, err
}
func (s *Store) SetPolicy(ctx context.Context, p c.Principal, v c.Policy) error {
	if p.Device != "" || p.Role != "owner" || p.Workspace == "" {
		return c.ErrForbidden
	}
	if v.Version < 1 || v.RetentionDays < 0 || (v.Content != "full" && v.Content != "metadata") || (v.Redaction != "none" && v.Redaction != "secrets") || (v.Visibility != "team" && v.Visibility != "self_managers") {
		return c.ErrInvalid
	}
	b, _ := json.Marshal(v)
	result, err := s.DB.Exec(ctx, `INSERT INTO tm_policies(workspace,version,body) VALUES($1,$2,$3)
 ON CONFLICT(workspace) DO UPDATE SET version=excluded.version,body=excluded.body WHERE tm_policies.version < excluded.version`, p.Workspace, v.Version, b)
	if err == nil && result.RowsAffected() == 0 {
		return c.ErrConflict
	}
	return err
}

func (s *Store) Accept(ctx context.Context, p c.Principal, v c.Snapshot) (c.Session, error) {
	empty := c.Session{}
	if p.Workspace == "" || p.Person == "" || p.Device == "" {
		return empty, c.ErrForbidden
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return empty, err
	}
	defer tx.Rollback(ctx)
	var pb []byte
	err = tx.QueryRow(ctx, "SELECT body FROM tm_policies WHERE workspace=$1 FOR SHARE", p.Workspace).Scan(&pb)
	if err != nil {
		return empty, c.ErrForbidden
	}
	var policy c.Policy
	if json.Unmarshal(pb, &policy) != nil {
		return empty, c.ErrInvalid
	}
	if v.PolicyVersion != policy.Version {
		return empty, c.ErrForbidden
	}
	v, b, err := applyContent(v, policy)
	if err != nil {
		return empty, err
	}
	var deleted bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM tm_deleted WHERE workspace=$1 AND source_ref=$2)", p.Workspace, v.SourceRef).Scan(&deleted)
	if err != nil {
		return empty, err
	}
	if deleted {
		return empty, c.ErrDeleted
	}
	attribution := "unknown_historical"
	started, e := time.Parse(time.RFC3339, v.StartedAt)
	if e == nil && !p.EnrolledAt.IsZero() && !started.Before(p.EnrolledAt) {
		attribution = "device_observed_not_verified_human"
	}
	digest := Hash(b)
	id := ID()
	_, err = tx.Exec(ctx, `INSERT INTO tm_snapshots(id,workspace,source_ref,revision,owner_id,device_id,attribution,digest,body) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(workspace,source_ref,revision) DO NOTHING`, id, p.Workspace, v.SourceRef, v.Revision, p.Person, p.Device, attribution, digest, b)
	if err != nil {
		return empty, err
	}
	var row c.Session
	var saved []byte
	var savedDigest string
	err = tx.QueryRow(ctx, `SELECT id,owner_id,device_id,attribution,received_at,digest,body FROM tm_snapshots WHERE workspace=$1 AND source_ref=$2 AND revision=$3`, p.Workspace, v.SourceRef, v.Revision).Scan(&row.ID, &row.Owner, &row.ObservedBy, &row.Attribution, &row.ReceivedAt, &savedDigest, &saved)
	if err != nil {
		return empty, err
	}
	if savedDigest != digest {
		return empty, c.ErrConflict
	}
	if err = json.Unmarshal(saved, &row.Snapshot); err != nil {
		return empty, err
	}
	if err = putMetrics(ctx, tx, row.ID, row.Snapshot); err != nil {
		return empty, err
	}
	if err = tx.Commit(ctx); err != nil {
		return empty, err
	}
	return row, nil
}
func (s *Store) Get(ctx context.Context, p c.Principal, id string) (c.Session, error) {
	var row c.Session
	var b []byte
	err := s.DB.QueryRow(ctx, `SELECT s.id,s.owner_id,s.device_id,s.attribution,s.received_at,s.body FROM tm_snapshots s JOIN tm_policies p ON p.workspace=s.workspace WHERE s.workspace=$1 AND s.id=$2 AND ($3 OR s.owner_id=$4 OR p.body->>'visibility'='team')`, p.Workspace, id, p.Manager(), p.Person).Scan(&row.ID, &row.Owner, &row.ObservedBy, &row.Attribution, &row.ReceivedAt, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return row, c.ErrNotFound
	}
	if err != nil {
		return row, err
	}
	if err = json.Unmarshal(b, &row.Snapshot); err != nil {
		return row, err
	}
	if err = s.Audit(ctx, p, "session.read", id); err != nil {
		return row, err
	}
	return row, nil
}
func (s *Store) List(ctx context.Context, p c.Principal, f c.Filter) ([]c.SessionSummary, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	before := f.Before
	if before.IsZero() {
		before = time.Now().Add(time.Minute)
	}
	pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(f.Query) + "%"
	rows, err := s.DB.Query(ctx, `SELECT s.id,s.owner_id,s.device_id,s.attribution,s.received_at,s.source_ref,s.revision,
 coalesce(s.body->>'client',''),coalesce(s.body->>'project',''),coalesce(s.body->>'branch',''),coalesce(s.body->>'started_at',''),
 coalesce((SELECT left(message->>'content',320) FROM jsonb_array_elements(s.body->'messages') message WHERE message->>'role'='user' LIMIT 1),''),
 jsonb_array_length(s.body->'messages'),coalesce(left(s.body->'raw'->>'display_name',200),''),coalesce(s.body->'raw'->>'is_automated'='true',false) FROM tm_snapshots s JOIN tm_policies p ON p.workspace=s.workspace
 WHERE s.workspace=$1 AND ($2 OR s.owner_id=$3 OR p.body->>'visibility'='team') AND ($4='' OR s.owner_id=$4) AND ($5='' OR s.body->>'project'=$5) AND ($6='' OR s.body->>'client'=$6) AND ($7='' OR s.body::text ILIKE $8) AND s.received_at<$9 AND ($11='' OR s.source_ref=$11) ORDER BY s.received_at DESC,s.id DESC LIMIT $10`, p.Workspace, p.Manager(), p.Person, f.Person, f.Project, f.Client, f.Query, pattern, before, f.Limit, f.SourceRef)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []c.SessionSummary{}
	for rows.Next() {
		var row c.SessionSummary
		if err = rows.Scan(&row.ID, &row.Owner, &row.ObservedBy, &row.Attribution, &row.ReceivedAt, &row.SourceRef, &row.Revision, &row.Client, &row.Project, &row.Branch, &row.StartedAt, &row.Preview, &row.MessageCount, &row.Title, &row.Automated); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if err = s.Audit(ctx, p, "session.list", "filtered"); err != nil {
		return nil, err
	}
	return out, nil
}
func (s *Store) Audit(ctx context.Context, p c.Principal, action, resource string) error {
	if p.Workspace == "" {
		return c.ErrForbidden
	}
	actor := p.Person
	if p.TokenKind == "debug" {
		actor = "agent:" + p.AgentLabel + " (issued by " + p.Person + ")"
	}
	_, err := s.DB.Exec(ctx, "INSERT INTO tm_audit(workspace,actor,action,resource) VALUES($1,$2,$3,$4)", p.Workspace, actor, action, resource)
	return err
}
func Wrap(err error) error { return fmt.Errorf("archive operation: %w", err) }

func applyContent(v c.Snapshot, p c.Policy) (c.Snapshot, []byte, error) { return policy.Apply(v, p) }
