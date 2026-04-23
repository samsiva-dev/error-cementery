package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Burial struct {
	ID        int64
	BuriedAt  time.Time
	ErrorText string
	ErrorHash string
	FixText   string
	Context   string
	Tags      string
	TimesDug  int
	LastDug   *time.Time
}

type BuryInput struct {
	ErrorText string
	FixText   string
	Context   string
	Tags      string
}

func (s *Store) Bury(input BuryInput) (*Burial, error) {
	hash := HashError(input.ErrorText)

	row := s.db.QueryRow(`SELECT id FROM burials WHERE error_hash = ?`, hash)
	var existingID int64
	if err := row.Scan(&existingID); err == nil {
		return nil, fmt.Errorf("already buried (id %d)", existingID)
	}

	res, err := s.db.Exec(`
		INSERT INTO burials (error_text, error_hash, fix_text, context, tags)
		VALUES (?, ?, ?, ?, ?)`,
		input.ErrorText, hash, input.FixText, input.Context, input.Tags,
	)
	if err != nil {
		return nil, fmt.Errorf("insert burial: %w", err)
	}

	id, _ := res.LastInsertId()
	return s.GetByID(id)
}

func (s *Store) GetByID(id int64) (*Burial, error) {
	row := s.db.QueryRow(`SELECT id, buried_at, error_text, error_hash, fix_text, context, tags, times_dug, last_dug FROM burials WHERE id = ?`, id)
	return scanBurial(row)
}

func (s *Store) GetByHash(hash string) (*Burial, error) {
	row := s.db.QueryRow(`SELECT id, buried_at, error_text, error_hash, fix_text, context, tags, times_dug, last_dug FROM burials WHERE error_hash = ?`, hash)
	return scanBurial(row)
}

func (s *Store) FTSSearch(query string, limit int) ([]Burial, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(`
		SELECT b.id, b.buried_at, b.error_text, b.error_hash, b.fix_text, b.context, b.tags, b.times_dug, b.last_dug
		FROM burial_fts
		JOIN burials b ON b.id = burial_fts.rowid
		WHERE burial_fts MATCH ?
		ORDER BY bm25(burial_fts) LIMIT ?`, query, limit)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	defer rows.Close()
	return scanBurials(rows)
}

func (s *Store) GetAll() ([]Burial, error) {
	rows, err := s.db.Query(`
		SELECT id, buried_at, error_text, error_hash, fix_text, context, tags, times_dug, last_dug
		FROM burials ORDER BY buried_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("get all: %w", err)
	}
	defer rows.Close()
	return scanBurials(rows)
}

func (s *Store) UpdateDigCount(id int64) error {
	_, err := s.db.Exec(`
		UPDATE burials SET times_dug = times_dug + 1, last_dug = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

func (s *Store) Update(id int64, input BuryInput) error {
	hash := HashError(input.ErrorText)
	_, err := s.db.Exec(`
		UPDATE burials SET error_text=?, error_hash=?, fix_text=?, context=?, tags=? WHERE id=?`,
		input.ErrorText, hash, input.FixText, input.Context, input.Tags, id)
	return err
}

func (s *Store) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM burials WHERE id = ?`, id)
	return err
}

func (s *Store) AllTags() ([]string, error) {
	rows, err := s.db.Query(`SELECT tags FROM burials WHERE tags != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[string]struct{}{}
	var result []string
	for rows.Next() {
		var tags string
		if err := rows.Scan(&tags); err != nil {
			continue
		}
		for _, t := range splitTags(tags) {
			if _, ok := seen[t]; !ok {
				seen[t] = struct{}{}
				result = append(result, t)
			}
		}
	}
	return result, nil
}

func (s *Store) Stats() (total int, topTags []TagCount, err error) {
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM burials`).Scan(&total); err != nil {
		return
	}

	rows, qErr := s.db.Query(`SELECT tags FROM burials WHERE tags != ''`)
	if qErr != nil {
		err = qErr
		return
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var tags string
		if sErr := rows.Scan(&tags); sErr != nil {
			continue
		}
		for _, t := range splitTags(tags) {
			counts[t]++
		}
	}

	for tag, count := range counts {
		topTags = append(topTags, TagCount{Tag: tag, Count: count})
	}
	sortTagCounts(topTags)
	if len(topTags) > 10 {
		topTags = topTags[:10]
	}
	return
}

type TagCount struct {
	Tag   string
	Count int
}

func sortTagCounts(tc []TagCount) {
	for i := 1; i < len(tc); i++ {
		for j := i; j > 0 && tc[j].Count > tc[j-1].Count; j-- {
			tc[j], tc[j-1] = tc[j-1], tc[j]
		}
	}
}

func (s *Store) SaveEmbedding(burialID int64, vector []float32) error {
	data, err := json.Marshal(vector)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		INSERT INTO embeddings(burial_id, vector_json) VALUES(?,?)
		ON CONFLICT(burial_id) DO UPDATE SET vector_json=excluded.vector_json`, burialID, string(data))
	return err
}

func (s *Store) LoadEmbeddings() (map[int64][]float32, error) {
	rows, err := s.db.Query(`SELECT burial_id, vector_json FROM embeddings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[int64][]float32{}
	for rows.Next() {
		var id int64
		var js string
		if err := rows.Scan(&id, &js); err != nil {
			continue
		}
		var v []float32
		if err := json.Unmarshal([]byte(js), &v); err != nil {
			continue
		}
		result[id] = v
	}
	return result, nil
}

func scanBurial(row *sql.Row) (*Burial, error) {
	var b Burial
	var lastDug sql.NullTime
	err := row.Scan(&b.ID, &b.BuriedAt, &b.ErrorText, &b.ErrorHash, &b.FixText, &b.Context, &b.Tags, &b.TimesDug, &lastDug)
	if err != nil {
		return nil, err
	}
	if lastDug.Valid {
		b.LastDug = &lastDug.Time
	}
	return &b, nil
}

func scanBurials(rows *sql.Rows) ([]Burial, error) {
	var burials []Burial
	for rows.Next() {
		var b Burial
		var lastDug sql.NullTime
		if err := rows.Scan(&b.ID, &b.BuriedAt, &b.ErrorText, &b.ErrorHash, &b.FixText, &b.Context, &b.Tags, &b.TimesDug, &lastDug); err != nil {
			return nil, err
		}
		if lastDug.Valid {
			b.LastDug = &lastDug.Time
		}
		burials = append(burials, b)
	}
	return burials, rows.Err()
}

func splitTags(tags string) []string {
	var result []string
	for _, t := range splitOn(tags, ',') {
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}

func splitOn(s string, sep rune) []string {
	var parts []string
	start := 0
	for i, r := range s {
		if r == sep {
			part := trim(s[start:i])
			parts = append(parts, part)
			start = i + 1
		}
	}
	parts = append(parts, trim(s[start:]))
	return parts
}

func trim(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// ── comments ──────────────────────────────────────────────────────────────────

type Comment struct {
	ID          int64
	BurialID    int64
	CommentText string
	CreatedAt   time.Time
}

func (s *Store) AddComment(burialID int64, text string) error {
	_, err := s.db.Exec(
		`INSERT INTO comments (burial_id, comment_text) VALUES (?, ?)`,
		burialID, text,
	)
	return err
}

func (s *Store) GetComments(burialID int64) ([]Comment, error) {
	rows, err := s.db.Query(
		`SELECT id, burial_id, comment_text, created_at FROM comments WHERE burial_id = ? ORDER BY created_at ASC`,
		burialID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.BurialID, &c.CommentText, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// GetAllComments returns all comments keyed by burial ID.
func (s *Store) GetAllComments() (map[int64][]Comment, error) {
	rows, err := s.db.Query(
		`SELECT id, burial_id, comment_text, created_at FROM comments ORDER BY burial_id, created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[int64][]Comment{}
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.BurialID, &c.CommentText, &c.CreatedAt); err != nil {
			return nil, err
		}
		result[c.BurialID] = append(result[c.BurialID], c)
	}
	return result, rows.Err()
}
