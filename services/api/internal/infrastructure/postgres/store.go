package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/eykh/restick/api_svc/internal/domain/apikey"
	"github.com/eykh/restick/api_svc/internal/domain/pack"
	"github.com/eykh/restick/api_svc/internal/domain/tombstone"
	"github.com/eykh/restick/api_svc/internal/domain/user"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Store struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	migFiles := []string{"0001_init.sql", "0002_add_sticker_keywords.sql", "0003_expand_api_keys.sql"}
	baseDirs := []string{"migrations", "api_svc/migrations", "/migrations"}

	for _, file := range migFiles {
		var sqlBytes []byte
		var err error
		for _, dir := range baseDirs {
			sqlBytes, err = os.ReadFile(dir + "/" + file)
			if err == nil {
				break
			}
		}
		if err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, string(sqlBytes)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Seed(ctx context.Context, adminEmail string, adminPassword string, apiKey string) error {
	var count int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return err
	}
	if count == 0 && adminEmail != "" && adminPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = s.pool.Exec(ctx, `INSERT INTO users (email, password_hash, display_name, role) VALUES ($1,$2,$3,$4)`,
			adminEmail, string(hash), "Admin", user.RoleAdmin)
		if err != nil {
			return err
		}
	}
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM api_keys").Scan(&count); err != nil {
		return err
	}
	if count == 0 && apiKey != "" {
		sum := sha256.Sum256([]byte(apiKey))
		prefix := "bootstrap"
		if len(apiKey) >= 12 {
			prefix = apiKey[:6] + "..." + apiKey[len(apiKey)-4:]
		}
		_, err := s.pool.Exec(ctx, `INSERT INTO api_keys (name, key_prefix, key_hash) VALUES ($1,$2,$3)`, "bootstrap", prefix, hex.EncodeToString(sum[:]))
		return err
	}
	return nil
}

func (s *Store) CreateUser(ctx context.Context, email string, passwordHash string, displayName string, role string) (user.User, error) {
	id := uuid.NewString()
	_, err := s.pool.Exec(ctx, `INSERT INTO users (id, email, password_hash, display_name, role) VALUES ($1,$2,$3,$4,$5)`,
		id, email, passwordHash, displayName, role)
	if err != nil {
		return user.User{}, err
	}
	return s.GetUserByEmail(ctx, email)
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (user.User, error) {
	var u user.User
	err := s.pool.QueryRow(ctx, `SELECT id::text, email, password_hash, display_name, role, created_at FROM users WHERE email=$1`, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, pack.ErrNotFound
	}
	return u, err
}

func (s *Store) GetUser(ctx context.Context, id string) (user.User, error) {
	var u user.User
	err := s.pool.QueryRow(ctx, `SELECT id::text, email, password_hash, display_name, role, created_at FROM users WHERE id=$1`, id).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, pack.ErrNotFound
	}
	return u, err
}

func (s *Store) CreateAPIKey(ctx context.Context, userID *string, name string, rawKey string, prefix string, keyHash string, expiresAt *time.Time) (apikey.CreatedAPIKey, error) {
	id := uuid.NewString()
	_, err := s.pool.Exec(ctx, `INSERT INTO api_keys (id, user_id, name, key_prefix, key_hash, is_active, expires_at)
		VALUES ($1, $2, $3, $4, $5, true, $6)`, id, userID, name, prefix, keyHash, expiresAt)
	if err != nil {
		return apikey.CreatedAPIKey{}, err
	}
	k, err := s.GetAPIKey(ctx, id)
	if err != nil {
		return apikey.CreatedAPIKey{}, err
	}
	return apikey.CreatedAPIKey{
		APIKey: k,
		RawKey: rawKey,
	}, nil
}

func (s *Store) GetAPIKey(ctx context.Context, id string) (apikey.APIKey, error) {
	var k apikey.APIKey
	var uID *string
	err := s.pool.QueryRow(ctx, `SELECT id::text, user_id::text, name, key_prefix, is_active, expires_at, last_used_at, created_at
		FROM api_keys WHERE id=$1`, id).
		Scan(&k.ID, &uID, &k.Name, &k.KeyPrefix, &k.IsActive, &k.ExpiresAt, &k.LastUsedAt, &k.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return apikey.APIKey{}, apikey.ErrNotFound
	}
	k.UserID = uID
	return k, err
}

func (s *Store) ListAPIKeysByUser(ctx context.Context, userID string) ([]apikey.APIKey, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text, user_id::text, name, key_prefix, is_active, expires_at, last_used_at, created_at
		FROM api_keys WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]apikey.APIKey, 0)
	for rows.Next() {
		var k apikey.APIKey
		var uID *string
		if err := rows.Scan(&k.ID, &uID, &k.Name, &k.KeyPrefix, &k.IsActive, &k.ExpiresAt, &k.LastUsedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		k.UserID = uID
		list = append(list, k)
	}
	return list, rows.Err()
}

func (s *Store) ListAllAPIKeys(ctx context.Context) ([]apikey.APIKey, error) {
	rows, err := s.pool.Query(ctx, `SELECT k.id::text, k.user_id::text, COALESCE(u.email, ''), k.name, k.key_prefix, k.is_active, k.expires_at, k.last_used_at, k.created_at
		FROM api_keys k
		LEFT JOIN users u ON k.user_id = u.id
		ORDER BY k.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]apikey.APIKey, 0)
	for rows.Next() {
		var k apikey.APIKey
		var uID *string
		if err := rows.Scan(&k.ID, &uID, &k.UserEmail, &k.Name, &k.KeyPrefix, &k.IsActive, &k.ExpiresAt, &k.LastUsedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		k.UserID = uID
		list = append(list, k)
	}
	return list, rows.Err()
}

func (s *Store) DeleteAPIKey(ctx context.Context, id string, userID string, isAdmin bool) error {
	var tag pgx.Rows
	_ = tag
	if isAdmin {
		res, err := s.pool.Exec(ctx, `DELETE FROM api_keys WHERE id=$1`, id)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return apikey.ErrNotFound
		}
		return nil
	}
	res, err := s.pool.Exec(ctx, `DELETE FROM api_keys WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return apikey.ErrNotFound
	}
	return nil
}

func (s *Store) RotateAPIKey(ctx context.Context, id string, userID string, isAdmin bool, newRawKey string, newPrefix string, newKeyHash string, newExpiresAt *time.Time) (apikey.CreatedAPIKey, error) {
	k, err := s.GetAPIKey(ctx, id)
	if err != nil {
		return apikey.CreatedAPIKey{}, err
	}
	if !isAdmin && (k.UserID == nil || *k.UserID != userID) {
		return apikey.CreatedAPIKey{}, apikey.ErrForbidden
	}
	_, err = s.pool.Exec(ctx, `UPDATE api_keys SET key_hash=$1, key_prefix=$2, expires_at=$3, last_used_at=NULL WHERE id=$4`,
		newKeyHash, newPrefix, newExpiresAt, id)
	if err != nil {
		return apikey.CreatedAPIKey{}, err
	}
	updated, err := s.GetAPIKey(ctx, id)
	if err != nil {
		return apikey.CreatedAPIKey{}, err
	}
	return apikey.CreatedAPIKey{
		APIKey: updated,
		RawKey: newRawKey,
	}, nil
}

func (s *Store) ValidateAPIKeyDetailed(ctx context.Context, raw string) (bool, string, string, string, error) {
	sum := sha256.Sum256([]byte(raw))
	hash := hex.EncodeToString(sum[:])
	var id string
	var uID *string
	var name string
	err := s.pool.QueryRow(ctx, `SELECT id::text, user_id::text, name FROM api_keys WHERE is_active AND key_hash=$1 AND (expires_at IS NULL OR expires_at > now())`, hash).
		Scan(&id, &uID, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", "", "", nil
	}
	if err != nil {
		return false, "", "", "", err
	}
	go func() {
		_, _ = s.pool.Exec(context.Background(), `UPDATE api_keys SET last_used_at=now() WHERE id=$1`, id)
	}()
	userIDStr := ""
	if uID != nil {
		userIDStr = *uID
	}
	return true, userIDStr, id, name, nil
}

func (s *Store) APIKeyValid(ctx context.Context, raw string) (bool, error) {
	valid, _, _, _, err := s.ValidateAPIKeyDetailed(ctx, raw)
	return valid, err
}

func (s *Store) CreateDraft(ctx context.Context, submitterID string, slug string, name string, author string, tags []string, sourceKind string) (pack.Pack, error) {
	id := uuid.NewString()
	prefix := "uploads/" + submitterID + "/" + id + "/"
	_, err := s.pool.Exec(ctx, `INSERT INTO packs (id, submitter_id, slug, name, author, tags, status, source_kind, source_prefix)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, submitterID, slug, name, author, tags, pack.StatusDraft, sourceKind, prefix)
	if err != nil {
		return pack.Pack{}, err
	}
	return s.GetPack(ctx, id)
}

func (s *Store) GetPack(ctx context.Context, id string) (pack.Pack, error) {
	var p pack.Pack
	err := s.pool.QueryRow(ctx, `SELECT id::text, submitter_id::text, slug, name, author, description, tags, current_version,
		checksum_sha256, thumbnail_key, bundle_key, sticker_count, ip_policy, status, reject_reason, source_kind, source_prefix
		FROM packs WHERE id=$1`, id).Scan(&p.ID, &p.SubmitterID, &p.Slug, &p.Name, &p.Author, &p.Description, &p.Tags,
		&p.CurrentVersion, &p.ChecksumSHA256, &p.ThumbnailKey, &p.BundleKey, &p.StickerCount, &p.IPPolicy, &p.Status,
		&p.RejectReason, &p.SourceKind, &p.SourcePrefix)
	if errors.Is(err, pgx.ErrNoRows) {
		return pack.Pack{}, pack.ErrNotFound
	}
	return p, err
}

func (s *Store) ListMine(ctx context.Context, submitterID string) ([]pack.Pack, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text, submitter_id::text, slug, name, author, description, tags, current_version,
		checksum_sha256, thumbnail_key, bundle_key, sticker_count, ip_policy, status, reject_reason, source_kind, source_prefix
		FROM packs WHERE submitter_id=$1 ORDER BY updated_at DESC`, submitterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPacks(rows)
}

func (s *Store) ListPublished(ctx context.Context, page int, pageSize int, tag string) ([]pack.Pack, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	var total int
	if tag == "" {
		if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM packs WHERE status=$1`, pack.StatusPublished).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err := s.pool.Query(ctx, `SELECT id::text, submitter_id::text, slug, name, author, description, tags, current_version,
			checksum_sha256, thumbnail_key, bundle_key, sticker_count, ip_policy, status, reject_reason, source_kind, source_prefix
			FROM packs WHERE status=$1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3`, pack.StatusPublished, pageSize, offset)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		items, err := scanPacks(rows)
		return items, total, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM packs WHERE status=$1 AND $2 = ANY(tags)`, pack.StatusPublished, tag).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text, submitter_id::text, slug, name, author, description, tags, current_version,
		checksum_sha256, thumbnail_key, bundle_key, sticker_count, ip_policy, status, reject_reason, source_kind, source_prefix
		FROM packs WHERE status=$1 AND $2 = ANY(tags) ORDER BY updated_at DESC LIMIT $3 OFFSET $4`, pack.StatusPublished, tag, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanPacks(rows)
	return items, total, err
}

func (s *Store) ListQueue(ctx context.Context, excludeUserID string) ([]pack.Pack, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text, submitter_id::text, slug, name, author, description, tags, current_version,
		checksum_sha256, thumbnail_key, bundle_key, sticker_count, ip_policy, status, reject_reason, source_kind, source_prefix
		FROM packs WHERE status=$1 AND submitter_id <> $2 ORDER BY updated_at ASC`, pack.StatusPendingReview, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPacks(rows)
}

func (s *Store) ListPublishedAdmin(ctx context.Context) ([]pack.Pack, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text, submitter_id::text, slug, name, author, description, tags, current_version,
		checksum_sha256, thumbnail_key, bundle_key, sticker_count, ip_policy, status, reject_reason, source_kind, source_prefix
		FROM packs WHERE status=$1 ORDER BY updated_at DESC`, pack.StatusPublished)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPacks(rows)
}

func scanPacks(rows pgx.Rows) ([]pack.Pack, error) {
	items := make([]pack.Pack, 0)
	for rows.Next() {
		var p pack.Pack
		if err := rows.Scan(&p.ID, &p.SubmitterID, &p.Slug, &p.Name, &p.Author, &p.Description, &p.Tags,
			&p.CurrentVersion, &p.ChecksumSHA256, &p.ThumbnailKey, &p.BundleKey, &p.StickerCount, &p.IPPolicy, &p.Status,
			&p.RejectReason, &p.SourceKind, &p.SourcePrefix); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

func (s *Store) Stickers(ctx context.Context, packID string) ([]pack.Sticker, error) {
	rows, err := s.pool.Query(ctx, `SELECT filename, emoji_hint, keywords, width, height, object_key, thumb_key, sort_order
		FROM stickers WHERE pack_id=$1 ORDER BY sort_order`, packID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]pack.Sticker, 0)
	for rows.Next() {
		var st pack.Sticker
		if err := rows.Scan(&st.Filename, &st.EmojiHint, &st.Keywords, &st.Width, &st.Height, &st.ObjectKey, &st.ThumbKey, &st.SortOrder); err != nil {
			return nil, err
		}
		items = append(items, st)
	}
	return items, rows.Err()
}

func (s *Store) EnqueueJob(ctx context.Context, p pack.Pack) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `UPDATE packs SET status=$1, updated_at=now() WHERE id=$2`, pack.StatusProcessing, p.ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO processing_jobs (pack_id, source_kind, source_prefix, status) VALUES ($1,$2,$3,'queued')`,
		p.ID, p.SourceKind, p.SourcePrefix)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type Job struct {
	ID           string
	PackID       string
	SourceKind   string
	SourcePrefix string
}

func (s *Store) ClaimJob(ctx context.Context) (Job, pack.Pack, user.User, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Job{}, pack.Pack{}, user.User{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var job Job
	err = tx.QueryRow(ctx, `SELECT id::text, pack_id::text, source_kind, source_prefix FROM processing_jobs
		WHERE status='queued' ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&job.ID, &job.PackID, &job.SourceKind, &job.SourcePrefix)
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, pack.Pack{}, user.User{}, false, nil
	}
	if err != nil {
		return Job{}, pack.Pack{}, user.User{}, false, err
	}
	_, err = tx.Exec(ctx, `UPDATE processing_jobs SET status='running' WHERE id=$1`, job.ID)
	if err != nil {
		return Job{}, pack.Pack{}, user.User{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Job{}, pack.Pack{}, user.User{}, false, err
	}
	p, err := s.GetPack(ctx, job.PackID)
	if err != nil {
		return Job{}, pack.Pack{}, user.User{}, false, err
	}
	u, err := s.GetUser(ctx, p.SubmitterID)
	return job, p, u, true, err
}

func (s *Store) CompleteJob(ctx context.Context, jobID string, checksum string, thumbKey string, bundleKey string, version int, stickers []pack.Sticker, nextStatus string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var packID string
	if err := tx.QueryRow(ctx, `UPDATE processing_jobs SET status='done' WHERE id=$1 RETURNING pack_id::text`, jobID).Scan(&packID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM stickers WHERE pack_id=$1`, packID)
	if err != nil {
		return err
	}
	for i, st := range stickers {
		kw := st.Keywords
		if kw == nil {
			kw = []string{}
		}
		_, err = tx.Exec(ctx, `INSERT INTO stickers (pack_id, filename, emoji_hint, keywords, width, height, object_key, thumb_key, sort_order)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, packID, st.Filename, st.EmojiHint, kw, st.Width, st.Height, st.ObjectKey, st.ThumbKey, i)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE packs SET status=$1, checksum_sha256=$2, thumbnail_key=$3, bundle_key=$4, current_version=$5,
		sticker_count=$6, updated_at=now() WHERE id=$7`, nextStatus, checksum, thumbKey, bundleKey, version, len(stickers), packID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) FailJob(ctx context.Context, jobID string, message string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var packID string
	if err := tx.QueryRow(ctx, `UPDATE processing_jobs SET status='failed', error=$1 WHERE id=$2 RETURNING pack_id::text`, message, jobID).Scan(&packID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE packs SET status=$1, reject_reason=$2, updated_at=now() WHERE id=$3`, pack.StatusRejected, message, packID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) SetPublished(ctx context.Context, packID string, reviewerID string, action string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `UPDATE packs SET status=$1, updated_at=now() WHERE id=$2`, pack.StatusPublished, packID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO pack_reviews (pack_id, reviewer_id, action) VALUES ($1,$2,$3)`, packID, reviewerID, action)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Reject(ctx context.Context, packID string, reviewerID string, reason pack.Reason) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `UPDATE packs SET status=$1, reject_reason=$2, updated_at=now() WHERE id=$3`, pack.StatusRejected, reason.Text, packID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO pack_reviews (pack_id, reviewer_id, action, reason_code, reason_text) VALUES ($1,$2,'reject',$3,$4)`,
		packID, reviewerID, reason.Code, reason.Text)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) DeleteRejected(ctx context.Context, packID string, submitterID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM packs WHERE id=$1 AND submitter_id=$2 AND status=$3`,
		packID, submitterID, pack.StatusRejected)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pack.ErrConflict
	}
	return nil
}

func (s *Store) Revoke(ctx context.Context, p pack.Pack, reviewerID string, reason pack.Reason) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `UPDATE packs SET status=$1, updated_at=now() WHERE id=$2`, pack.StatusRevoked, p.ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO pack_reviews (pack_id, reviewer_id, action, reason_code, reason_text) VALUES ($1,$2,'revoke',$3,$4)`,
		p.ID, reviewerID, reason.Code, reason.Text)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO pack_tombstones (pack_id, revoked_by, reason_code, reason_text, checksum_sha256)
		VALUES ($1,$2,$3,$4,$5) ON CONFLICT (pack_id) DO UPDATE SET revoked_at=now(), reason_code=$3, reason_text=$4, checksum_sha256=$5`,
		p.ID, reviewerID, reason.Code, reason.Text, p.ChecksumSHA256)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Tombstones(ctx context.Context, since time.Time) ([]tombstone.Tombstone, error) {
	rows, err := s.pool.Query(ctx, `SELECT pack_id::text, revoked_at, reason_code, checksum_sha256 FROM pack_tombstones
		WHERE revoked_at >= $1 ORDER BY revoked_at ASC LIMIT 1000`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]tombstone.Tombstone, 0)
	for rows.Next() {
		var t tombstone.Tombstone
		if err := rows.Scan(&t.PackID, &t.RevokedAt, &t.ReasonCode, &t.ChecksumSHA256); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

func (s *Store) UpsertCLI(ctx context.Context, p pack.Pack, stickers []pack.Sticker) error {
	adminID, err := s.firstAdminID(ctx)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO packs (id, submitter_id, slug, name, author, tags, current_version, checksum_sha256,
		thumbnail_key, bundle_key, sticker_count, status, source_kind, source_prefix)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'files','')
		ON CONFLICT (id) DO UPDATE SET checksum_sha256=EXCLUDED.checksum_sha256, thumbnail_key=EXCLUDED.thumbnail_key,
		bundle_key=EXCLUDED.bundle_key, sticker_count=EXCLUDED.sticker_count, current_version=EXCLUDED.current_version,
		status=$12, updated_at=now()`, p.ID, adminID, p.Slug, p.Name, p.Author, p.Tags, p.CurrentVersion, p.ChecksumSHA256,
		p.ThumbnailKey, p.BundleKey, len(stickers), pack.StatusPendingReview)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM stickers WHERE pack_id=$1`, p.ID)
	if err != nil {
		return err
	}
	for i, st := range stickers {
		kw := st.Keywords
		if kw == nil {
			kw = []string{}
		}
		_, err = tx.Exec(ctx, `INSERT INTO stickers (pack_id, filename, emoji_hint, keywords, width, height, object_key, thumb_key, sort_order)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, p.ID, st.Filename, st.EmojiHint, kw, st.Width, st.Height, st.ObjectKey, st.ThumbKey, i)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) PackUserForJob(ctx context.Context, jobID string) (pack.Pack, user.User, error) {
	var packID string
	err := s.pool.QueryRow(ctx, `SELECT pack_id::text FROM processing_jobs WHERE id=$1`, jobID).Scan(&packID)
	if err != nil {
		return pack.Pack{}, user.User{}, err
	}
	p, err := s.GetPack(ctx, packID)
	if err != nil {
		return pack.Pack{}, user.User{}, err
	}
	u, err := s.GetUser(ctx, p.SubmitterID)
	return p, u, err
}

func (s *Store) firstAdminID(ctx context.Context) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id::text FROM users WHERE role=$1 ORDER BY created_at LIMIT 1`, user.RoleAdmin).Scan(&id)
	return id, err
}

type RejectionStat struct {
	ReasonCode     string    `json:"reason_code"`
	RejectionCount int       `json:"rejection_count"`
	LastRejectedAt time.Time `json:"last_rejected_at"`
}

func (s *Store) RejectionAnalytics(ctx context.Context) ([]RejectionStat, error) {
	rows, err := s.pool.Query(ctx, `SELECT reason_code, rejection_count, last_rejected_at FROM view_rejection_analytics ORDER BY rejection_count DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]RejectionStat, 0)
	for rows.Next() {
		var st RejectionStat
		if err := rows.Scan(&st.ReasonCode, &st.RejectionCount, &st.LastRejectedAt); err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	return stats, rows.Err()
}

func (s *Store) IsSlugTaken(ctx context.Context, slug string, excludePackID string) (bool, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM packs WHERE slug=$1 AND id<>$2`, slug, excludePackID).Scan(&count)
	return count > 0, err
}

func (s *Store) UpdatePackInfo(ctx context.Context, packID string, name string, slug string, tags []string) (pack.Pack, error) {
	if tags == nil {
		tags = []string{}
	}
	_, err := s.pool.Exec(ctx, `UPDATE packs SET name=$1, slug=$2, tags=$3, updated_at=now() WHERE id=$4`, name, slug, tags, packID)
	if err != nil {
		return pack.Pack{}, err
	}
	return s.GetPack(ctx, packID)
}

func (s *Store) UpdateStickersMetadata(ctx context.Context, packID string, items []pack.Sticker) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, st := range items {
		kw := st.Keywords
		if kw == nil {
			kw = []string{}
		}
		_, err = tx.Exec(ctx, `UPDATE stickers SET emoji_hint=$1, keywords=$2 WHERE pack_id=$3 AND filename=$4`,
			st.EmojiHint, kw, packID, st.Filename)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

