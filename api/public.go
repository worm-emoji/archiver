package api

import (
	"database/sql"
	"net/http"
	"time"
)

type PublicBookmark struct {
	URL   string    `json:"url"`
	Title string    `json:"title"`
	Time  time.Time `json:"time"`
}

func (s *Server) getPublicBookmarks(w http.ResponseWriter, r *http.Request) {
	const query = `
		select url, title, ts
		from bookmarks
		where tags @> '{publish}'::text[]
		order by ts desc;
	`

	var (
		ctx   = r.Context()
		url   string
		title sql.NullString
		ts    time.Time
		resp  = []PublicBookmark{}
	)

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		s.sendJSONError(r, w, err, http.StatusInternalServerError, "error getting public bookmarks")
		return
	}

	for rows.Next() {
		if err := rows.Scan(&url, &title, &ts); err != nil {
			s.sendJSONError(r, w, err, http.StatusInternalServerError, "error getting public bookmarks")
			return
		}

		resp = append(resp, PublicBookmark{
			URL:   url,
			Title: title.String,
			Time:  ts,
		})
	}

	s.sendJSON(r, w, resp)
}
