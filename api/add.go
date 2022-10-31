package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type Bookmark struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Time        time.Time `json:"time"`
}

type AddRequest struct {
	Bookmarks []Bookmark `json:"bookmarks"`
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

func (s *Server) add(w http.ResponseWriter, r *http.Request) {
	var (
		req AddRequest
		ctx = r.Context()
	)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSONError(r, w, err, http.StatusBadRequest, "invalid request")
		return
	}
	defer r.Body.Close()

	if len(req.Bookmarks) == 0 {
		s.sendJSONError(r, w, nil, http.StatusBadRequest, "no bookmarks provided")
		return
	}

	const addQuery = `
	INSERT INTO bookmarks (url, title, description, tags, ts)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT(url) DO NOTHING;`

	for _, b := range req.Bookmarks {

		t := b.Time
		if t.IsZero() {
			t = time.Now()
		}

		// sometimes shortcuts send in titles that are urls, i'd
		// rather them be null
		if b.Title == b.URL {
			b.Title = ""
		}

		// no tags? null
		if len(b.Tags) == 0 {
			b.Tags = nil
		}

		_, err := s.db.Exec(
			ctx, addQuery,
			b.URL, nullStr(b.Title), nullStr(b.Description), b.Tags, t,
		)

		if err != nil {
			s.sendJSONError(r, w, err, http.StatusInternalServerError, "failed to add bookmark")
			return
		}
	}

	s.sendJSON(r, w, map[string]any{
		"status": "ok",
	})
}
