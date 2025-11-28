package app

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	"sqlite-gui/pkg/database/sqlite"
)

var (
	port   = flag.Int("port", 3000, "The server port")
	dbPath = flag.String("db", "file:sqlite-gui.db?_pragma=foreign_keys(1)", "SQLite connection string")
)

func Run() {
	flag.Parse()

	ctx := context.Background()
	db := sqlite.New()
	if err := db.Connect(ctx, *dbPath); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	api := NewAPI(db)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})
	api.RegisterRoutes(mux)

	log.Printf("Starting server on port %d", *port)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Fatal(http.Serve(lis, mux))
}
