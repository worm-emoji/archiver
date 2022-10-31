package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

type Server struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Server {
	return &Server{
		db: db,
	}
}

func (s *Server) Handler(env, gitSha string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, gitSha)
	})
	mux.HandleFunc("/api/bookmark", s.withAuth(s.addBookmark))
	mux.HandleFunc("/api/crawl", s.withAuth(s.addCrawl))
	mux.HandleFunc("/api/crawl/pending", s.withAuth(s.getPendingCrawls))

	h := http.Handler(mux)
	h = versionHandler(h, gitSha)
	h = hlog.UserAgentHandler("user_agent")(h)
	h = hlog.RefererHandler("referer")(h)
	h = hlog.RequestIDHandler("req_id", "Request-Id")(h)
	h = hlog.URLHandler("path")(h)
	h = hlog.MethodHandler("method")(h)
	h = requestLogger(gitSha, h)
	h = RemoteAddrHandler("ip")(h)
	h = hlog.NewHandler(log.Logger)(h) // needs to be last for log values to correctly be passed to context

	if env == "production" {
		return h
	}

	c := cors.New(cors.Options{
		AllowedOrigins:     []string{"*"},
		AllowCredentials:   false,
		OptionsPassthrough: false,
	})

	h = c.Handler(h)

	return h
}

func ipFromRequest(r *http.Request) string {
	if r.Header.Get("x-forwarded-for") != "" {
		group := strings.Split(r.Header.Get("x-forwarded-for"), ", ")
		if len(group) > 0 {
			return group[len(group)-1]
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return ""
}

func RemoteAddrHandler(fieldKey string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ipFromRequest(r)
			if ip != "" {
				log := zerolog.Ctx(r.Context())
				log.UpdateContext(func(c zerolog.Context) zerolog.Context {
					return c.Str("ip", ip)
				})
			}

			next.ServeHTTP(w, r.WithContext(r.Context()))
		})
	}
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		key := strings.TrimLeft(r.Header.Get("Authorization"), "Bearer ")
		if key == "" {
			s.sendJSONError(r, w, nil, http.StatusUnauthorized, "missing auth token")
			return
		}

		var isValid bool

		const query = `
			SELECT EXISTS (
				SELECT 1
				FROM api_keys
				WHERE key = $1
			);
		`
		err := s.db.QueryRow(r.Context(), query, key).Scan(&isValid)

		if err != nil {
			s.sendJSONError(r, w, err, http.StatusInternalServerError, "error checking auth token")
			return
		}

		if !isValid {
			s.sendJSONError(r, w, nil, http.StatusUnauthorized, "invalid auth token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func versionHandler(h http.Handler, sha string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("server-version", sha)
		h.ServeHTTP(w, r)
	})
}

func requestLogger(sha string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			h.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		log := zerolog.Ctx(ctx)

		sc := &statusCapture{ResponseWriter: w}

		requestStart := time.Now()
		h.ServeHTTP(sc, r.WithContext(ctx))

		// log every request
		log.Info().
			Int("status", sc.status).
			Dur("duration", time.Since(requestStart)).
			Msg("")
	})
}

type statusCapture struct {
	http.ResponseWriter
	wroteHeader bool
	status      int
}

func (s *statusCapture) WriteHeader(c int) {
	s.status = c
	s.wroteHeader = true
	s.ResponseWriter.WriteHeader(c)
}

func (s *statusCapture) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.WriteHeader(http.StatusOK)
	}
	return s.ResponseWriter.Write(b)
}

func (s *Server) sendJSON(r *http.Request, w http.ResponseWriter, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) sendJSONError(
	r *http.Request,
	w http.ResponseWriter,
	err error,
	code int,
	customMessage string,
) {
	w.Header().Set("Content-Type", "application/json")
	if code == http.StatusNotFound && w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
	}

	// all headers need to be set before this line
	w.WriteHeader(code)

	if err != nil {
		log.Ctx(r.Context()).Err(err).Send()
	}

	message := http.StatusText(code)
	if customMessage != "" {
		message = customMessage
	}

	json.NewEncoder(w).Encode(map[string]any{
		"error":   true,
		"message": message,
	})
}
