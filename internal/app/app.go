package app

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"sqlite-gui/pkg/database"
	"sqlite-gui/pkg/database/sqlite"
)

const defaultConnectionString = "main=file:sqlite-gui.db?_pragma=foreign_keys(1)"

var (
	port    = flag.Int("port", 3000, "The server port")
	dbPaths dbFlag
)

type dbFlag []string

func (f *dbFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *dbFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func init() {
	flag.Var(&dbPaths, "db", "SQLite connection string (repeatable). Format name=connectionString to label the connection.")
}

func Run() {
	flag.Parse()
	if len(dbPaths) == 0 {
		dbPaths = append(dbPaths, defaultConnectionString)
	}

	ctx := context.Background()
	manager := NewConnectionManager(func() database.Database { return sqlite.New() })
	for i, raw := range dbPaths {
		name, conn := parseConnectionArg(raw, fmt.Sprintf("db%d", i+1))
		if err := manager.Add(ctx, name, conn); err != nil {
			log.Fatalf("failed to connect to database %q: %v", conn, err)
		}
		log.Printf("Connected to %q (%s)", name, conn)
	}
	defer manager.CloseAll()

	api := NewAPI(manager)

	/*
	 * ROUTES DEFINITION START
	 */
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})
	api.RegisterRoutes(mux)
	handler := corsMiddleware(mux)
	/*
	 * ROUTES DEFINITION END
	 */

	log.Printf("Starting server on port %d", *port)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Fatal(http.Serve(lis, handler))
}

func parseConnectionArg(raw, fallbackName string) (string, string) {
	if parts := strings.SplitN(raw, "=", 2); len(parts) == 2 && strings.TrimSpace(parts[0]) != "" {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}

	name := fallbackName
	if derived := deriveName(raw); derived != "" {
		name = derived
	}
	return name, raw
}

func deriveName(raw string) string {
	conn := strings.TrimPrefix(raw, "file:")
	if idx := strings.Index(conn, "?"); idx != -1 {
		conn = conn[:idx]
	}
	base := filepath.Base(conn)
	base = strings.TrimSuffix(base, ".db")
	return strings.TrimSpace(base)
}
