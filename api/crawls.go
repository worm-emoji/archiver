package api

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx"
)

type PendingCrawlsResponse struct {
	URLs []string `json:"urls"`
}

func (s *Server) getPendingCrawls(w http.ResponseWriter, r *http.Request) {
	const query = `
    with pending as (
		select url
		from bookmarks
		where url not in (
			select url from crawls
			where crawl_attempt is null or
				crawl_attempt > now() - interval '1 day'
		)
		limit 5
		for update skip locked
	)

	update bookmarks
	set crawl_attempt = now()
	where url in (select url from pending)
	returning url;
	`

	var (
		ctx  = r.Context()
		resp = PendingCrawlsResponse{}
	)

	rows, err := s.db.Query(ctx, query)
	if errors.Is(err, pgx.ErrNoRows) {
		s.sendJSON(r, w, resp)
		return
	} else if err != nil {
		s.sendJSONError(r, w, err, http.StatusInternalServerError, "error getting pending crawls")
		return
	}

	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			s.sendJSONError(r, w, err, http.StatusInternalServerError, "error getting pending crawls")
			return
		}

		resp.URLs = append(resp.URLs, url)
	}

	s.sendJSON(r, w, resp)
}
