package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/saintedlama/degubase/internal/automation"
	"github.com/saintedlama/degubase/internal/identity"
	"github.com/saintedlama/degubase/internal/infrastructure/events"
	"github.com/saintedlama/degubase/internal/infrastructure/health"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/infrastructure/storage"
	"github.com/saintedlama/degubase/internal/jobs"
	"github.com/saintedlama/degubase/internal/models"
	"github.com/saintedlama/degubase/internal/records"
	"github.com/saintedlama/degubase/internal/schema"
	"github.com/saintedlama/degubase/internal/skills"
	"github.com/saintedlama/degubase/internal/snapshots"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

func withWorkspace(s identity.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ws, err := s.GetWorkspaceByCode(r.Context(), chi.URLParam(r, "wsCode"))
			if err != nil {
				httplib.InternalErr(w, err)
				return
			}
			if ws == nil {
				httplib.NotFound(w)
				return
			}
			// API token auth is workspace-scoped; reject if token belongs to a different workspace
			if tokenWsID, ok := httplib.TokenWsIDFromCtx(r); ok && ws.ID != tokenWsID {
				httplib.Forbidden(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), httplib.CtxWorkspace, ws)))
		})
	}
}

func withTable(s schema.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ws := r.Context().Value(httplib.CtxWorkspace).(*models.Workspace)
			tbl, err := s.GetTableByCode(r.Context(), ws.ID, chi.URLParam(r, "tableCode"))
			if err != nil {
				httplib.InternalErr(w, err)
				return
			}
			if tbl == nil {
				httplib.NotFound(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), httplib.CtxTable, tbl)))
		})
	}
}

func NewRouter(db *sql.DB, fileStore storage.Storage, uploadCfg records.UploadConfig, uiDir string, jwtSecret []byte, disableAuth bool, snap *snapshots.Handler, jobs *jobs.Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(serviceDesc)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	broker := events.NewBroker()

	var ident identity.Store
	var schem schema.Store
	var recs records.Store
	var auto automation.Store
	var sklStore skills.Store
	ident = identity.NewSQLite(db)
	schem = schema.NewSQLite(db)
	recs = records.NewSQLite(db)
	auto = automation.NewSQLite(db)
	sklStore = skills.NewSQLite(db)

	ws := &identity.WorkspaceHandler{Store: ident}
	tbl := &schema.TableHandler{Store: schem}
	col := &schema.ColumnHandler{Store: schem}
	fl := &records.FileHandler{Rows: recs, Cols: schem, Storage: fileStore, Config: uploadCfg}
	rowSvc := &records.RowService{Store: recs, Cols: schem}
	row := &records.RowHandler{Store: recs, Cols: schem, Svc: rowSvc, Broker: broker, Files: fl, Tbls: schem}
	csvh := &records.CSVHandler{Store: recs, Cols: schem, Svc: rowSvc}
	gph := &records.GraphHandler{Svc: rowSvc, Tbls: schem}
	refh := &records.ReferencingHandler{Store: recs, Cols: schem, Svc: rowSvc, Tbls: schem}
	vw := &schema.ViewHandler{Store: schem}
	auth := &identity.AuthHandler{Store: ident, JWTSecret: jwtSecret, DisableAuth: disableAuth}
	tok := &identity.TokenHandler{Store: ident}
	evh := &records.EventHandler{Broker: broker}
	scr := &automation.ScriptHandler{Store: auto}
	skl := &skills.SkillHandler{Store: sklStore, Tbls: schem, Rows: recs, Ident: ident, DisableAuth: disableAuth}

	// Wire the script runner: dispatch Lua scripts on row events.
	runner := automation.NewScriptRunner(context.Background(), auto, rowMutatorAdapter{s: recs, cols: schem}, schem, broker)
	go func() {
		for ev := range broker.SubscribeAll() {
			if strings.HasPrefix(ev.CommandSourceID, "script-") {
				continue // skip script-originated events to prevent trigger loops
			}
			table, err := schem.GetTable(context.Background(), ev.TableID)
			if err != nil || table == nil {
				continue
			}
			workspace, err := ident.GetWorkspace(context.Background(), table.WorkspaceID)
			if err != nil || workspace == nil {
				continue
			}
			var scriptEvent models.ScriptEventType
			switch ev.Type {
			case events.RowEventCreated:
				scriptEvent = models.ScriptEventRecordCreated
			case events.RowEventUpdated:
				scriptEvent = models.ScriptEventRecordUpdated
			case events.RowEventDeleted:
				scriptEvent = models.ScriptEventRecordDeleted
			default:
				continue
			}
			runner.Dispatch(context.Background(), workspace, table, scriptEvent, ev.Row)
		}
	}()

	var authMw func(http.Handler) http.Handler
	var adminMw func(http.Handler) http.Handler

	if disableAuth {
		auth.DefaultUser = ensureDefaultUser(ident)
		authMw = withDefaultUser(auth.DefaultUser)
		adminMw = func(next http.Handler) http.Handler { return next }
	} else {
		authMw = requireAuth(ident, jwtSecret)
		adminMw = requireAdmin
	}

	r.Get("/api/health", health.Handler)

	r.Get("/api/doc", apiDocsLanding)
	r.Get("/api/docs", apiDocsLanding)
	r.Get("/api/docs/*", httpSwagger.WrapHandler)

	// Auth (public)
	r.Route("/api/auth", func(r chi.Router) {
		r.Get("/setup", auth.SetupStatus)
		r.Post("/login", auth.Login)
		r.Post("/logout", auth.Logout)
		r.With(authMw).Get("/me", auth.Me)
	})

	// Admin routes — authMw and adminMw are no-ops when auth is disabled.
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(authMw, adminMw)
		r.Get("/users", auth.ListUsers)
		r.Post("/users", auth.CreateUser)
		r.Delete("/users/{userID}", auth.DeleteUser)
		if snap != nil {
			r.Get("/snapshots", snap.ListSnapshots)
			r.Post("/snapshots", snap.CreateSnapshot)
			r.Post("/snapshots/{id}/restore", snap.ScheduleRestore)
		}
		if jobs != nil {
			r.Get("/jobs", jobs.ListJobRuns)
		}
	})

	r.With(authMw).Route("/api/workspaces", func(r chi.Router) {
		r.Get("/", ws.List)
		r.Post("/", ws.Create)
		r.Route("/{wsCode}", func(r chi.Router) {
			r.Use(withWorkspace(ident))
			r.Get("/", ws.Get)
			r.Put("/", ws.Update)
			r.Delete("/", ws.Delete)

			if !disableAuth {
				r.Route("/tokens", func(r chi.Router) {
					r.Get("/", tok.List)
					r.Post("/", tok.Create)
					r.Delete("/{tokenID}", tok.Delete)
				})
			}

			r.Route("/tables", func(r chi.Router) {
				r.Get("/", tbl.List)
				r.Post("/", tbl.Create)
				r.Route("/{tableCode}", func(r chi.Router) {
					r.Use(withTable(schem))
					r.Get("/", tbl.Get)
					r.Put("/", tbl.Update)
					r.Delete("/", tbl.Delete)

					r.Route("/columns", func(r chi.Router) {
						r.Get("/", col.List)
						r.Post("/", col.Create)
						r.Put("/reorder", col.Reorder)
						r.Put("/{colCode}", col.Update)
						r.Delete("/{colCode}", col.Delete)
					})

					r.Route("/rows", func(r chi.Router) {
						r.Get("/", row.List)
						r.Post("/", row.Create)
						r.Post("/bulk", row.BulkPatch)
						r.Get("/export.csv", csvh.Export)
						r.Post("/import", csvh.Import)
						r.Get("/{rowID}", row.Get)
						r.Put("/{rowID}", row.Update)
						r.Patch("/{rowID}", row.Patch)
						r.Delete("/{rowID}", row.Delete)
						r.Get("/{rowID}/graph", gph.Graph)
						r.Get("/{rowID}/referencing", refh.ListReferencing)
						r.Get("/{rowID}/history", row.History)
						r.Post("/{rowID}/history", row.CreateAnnotation)
						r.Patch("/{rowID}/history/{histID}", row.UpdateHistoryAnnotation)
						r.Delete("/{rowID}/history/{histID}", row.DeleteAnnotation)
						r.Post("/{rowID}/files", fl.Upload)
						r.Get("/{rowID}/files/{fileID}", fl.ServeFile)
						r.Get("/{rowID}/files/{fileID}/thumbnail", fl.ServeThumbnail)
					})

					r.Get("/groups", row.ListGroups)
					r.Get("/events", evh.Stream)

					r.Route("/views", func(r chi.Router) {
						r.Get("/", vw.List)
						r.Post("/", vw.Create)
						r.Get("/{viewCode}", vw.Get)
						r.Put("/{viewCode}", vw.Update)
						r.Delete("/{viewCode}", vw.Delete)
					})
				})
			})

			r.Route("/scripts", func(r chi.Router) {
				r.Get("/", scr.List)
				r.Post("/", scr.Create)
				r.Get("/env", scr.ListEnv)
				r.Post("/env", scr.UpsertEnv)
				r.Delete("/env/{envID}", scr.DeleteEnv)
				r.Get("/{scriptID}", scr.Get)
				r.Put("/{scriptID}", scr.Update)
				r.Delete("/{scriptID}", scr.Delete)
				r.Get("/{scriptID}/executions", scr.ListExecutions)
				r.Post("/{scriptID}/executions", scr.CreateExecution)
			})

			r.Route("/skills", func(r chi.Router) {
				r.Get("/", skl.List)
				r.Post("/", skl.Create)
				r.Post("/preview", skl.Preview)
				r.Get("/{skillID}", skl.Get)
				r.Patch("/{skillID}", skl.Update)
				r.Delete("/{skillID}", skl.Delete)
				r.Get("/{skillID}/skill.md", skl.ServeSkillMd)
				r.Get("/{skillID}/context", skl.ServeContext)
			})
		})
	})

	if uiDir != "" {
		r.Handle("/*", spaHandler(uiDir))
	}

	return r
}

// rowMutatorAdapter adapts records.Store to the automation.RowMutator interface,
// dropping the revisionID that the automation runner doesn't need.
type rowMutatorAdapter struct {
	s    records.Store
	cols automation.ColumnLister
}

func (a rowMutatorAdapter) GetRow(ctx context.Context, id int64) (*models.Row, error) {
	return a.s.GetRow(ctx, id)
}

func (a rowMutatorAdapter) CreateRow(ctx context.Context, tableID int64, data json.RawMessage) (*models.Row, error) {
	cols, err := a.cols.ListColumns(ctx, tableID)
	if err != nil {
		return nil, err
	}
	return a.s.CreateRow(ctx, tableID, data, cols)
}

func (a rowMutatorAdapter) UpdateRow(ctx context.Context, id int64, data json.RawMessage) (*models.Row, error) {
	row, err := a.s.GetRow(ctx, id)
	if err != nil || row == nil {
		return row, err
	}
	cols, err := a.cols.ListColumns(ctx, row.TableID)
	if err != nil {
		return nil, err
	}
	updated, _, err := a.s.UpdateRow(ctx, id, data, "", cols)
	return updated, err
}

func serviceDesc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `</api/docs/swagger.json>; rel="service-desc"`)
		next.ServeHTTP(w, r)
	})
}

func apiDocsLanding(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>DeguBase API Docs</title>
  <link rel="service-desc" href="/api/docs/swagger.json" type="application/json">
</head>
<body>
  <h1>DeguBase API</h1>
  <ul>
    <li><a href="/api/docs/index.html">Swagger UI</a></li>
    <li><a href="/api/docs/swagger.json">OpenAPI spec (JSON)</a></li>
    <li><a href="/api/docs/swagger.yaml">OpenAPI spec (YAML)</a></li>
  </ul>
</body>
</html>`))
}

func spaHandler(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		fullPath := filepath.Join(dir, filepath.FromSlash(path.Clean("/"+strings.TrimPrefix(r.URL.Path, "/"))))
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
