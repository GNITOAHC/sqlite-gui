package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"sqlite-gui/pkg/database"
)

type API struct {
	connections *ConnectionManager
}

func NewAPI(connections *ConnectionManager) *API {
	return &API{connections: connections}
}

func (api *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/connections", api.listConnections)
	mux.HandleFunc("POST /api/connections", api.addConnection)
	mux.HandleFunc("GET /api/tables", api.listTables)
	mux.HandleFunc("GET /api/tables/{table}/columns", api.getColumns)
	mux.HandleFunc("GET /api/tables/{table}/rows", api.getRows)
	mux.HandleFunc("POST /api/tables/{table}/rows", api.insertRow)
	mux.HandleFunc("PUT /api/tables/{table}/rows/{id}", api.updateRow)
	mux.HandleFunc("DELETE /api/tables/{table}/rows/{id}", api.deleteRow)
	mux.HandleFunc("POST /api/query", api.query)
	mux.HandleFunc("POST /api/exec", api.exec)
}

// listConnections returns all known database connections.
// curl: curl -X GET http://localhost:3000/api/connections
func (api *API) listConnections(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"connections": api.connections.List(),
		"default":     api.connections.Default(),
	})
}

// addConnection adds a new database connection using the given name/connection string.
// curl: curl -X POST -H "Content-Type: application/json" -d '{"name":"main","connString":":memory:"}' http://localhost:3000/api/connections
func (api *API) addConnection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		ConnString string `json:"connString"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.ConnString) == "" {
		writeError(w, http.StatusBadRequest, errors.New("connString is required"))
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = deriveName(req.ConnString)
	}
	if name == "" {
		name = fmt.Sprintf("db%d", len(api.connections.List())+1)
	}
	if err := api.connections.Add(r.Context(), name, req.ConnString); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrConnectionExists) {
			status = http.StatusConflict
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"connection": ConnectionInfo{
			Name:       name,
			ConnString: req.ConnString,
			Default:    api.connections.Default() == name,
		},
	})
}

// listTables returns table names for the selected database (?db=name is optional).
// curl: curl -X GET "http://localhost:3000/api/tables?db=db1"
func (api *API) listTables(w http.ResponseWriter, r *http.Request) {
	db, ok := api.useDB(w, r)
	if !ok {
		return
	}
	tables, err := db.Tables(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tables": tables})
}

// getColumns returns column definitions for a table.
// curl: curl -X GET "http://localhost:3000/api/tables/users/columns?db=db1"
func (api *API) getColumns(w http.ResponseWriter, r *http.Request) {
	db, ok := api.useDB(w, r)
	if !ok {
		return
	}
	table := r.PathValue("table")
	cols, err := db.Columns(r.Context(), table)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"columns": cols})
}

// getRows returns rows for a table with optional limit/offset.
// curl: curl -X GET "http://localhost:3000/api/tables/users/rows?limit=10&offset=0&db=db1"
func (api *API) getRows(w http.ResponseWriter, r *http.Request) {
	db, ok := api.useDB(w, r)
	if !ok {
		return
	}
	table := r.PathValue("table")
	limit := queryInt(r, "limit")
	offset := queryInt(r, "offset")
	rows, err := db.Rows(r.Context(), table, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

// insertRow inserts a JSON row into the given table.
// curl: curl -X POST -H "Content-Type: application/json" -d '{"name":"alice","age":30}' "http://localhost:3000/api/tables/users/rows?db=db1"
func (api *API) insertRow(w http.ResponseWriter, r *http.Request) {
	db, ok := api.useDB(w, r)
	if !ok {
		return
	}
	table := r.PathValue("table")
	row, err := decodeRow(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := db.Insert(r.Context(), table, row); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

// updateRow updates a row by primary key column/value (supports composite keys).
//
//	curl: curl -X PUT -H "Content-Type: application/json" \
//	  -d '{"role":"admin"}' \
//	  "http://localhost:3000/api/tables/memberships/rows/1,2?pk=user_id,team_id&db=db1"
func (api *API) updateRow(w http.ResponseWriter, r *http.Request) {
	db, ok := api.useDB(w, r)
	if !ok {
		return
	}
	table := r.PathValue("table")
	id := parsePathID(r.PathValue("id"))
	key, err := buildKey(primaryKeyColumns(r.URL.Query().Get("pk")), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	row, err := decodeRow(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := db.Update(r.Context(), table, key, row); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// deleteRow deletes a row by primary key column/value (supports composite keys).
// curl: curl -X DELETE "http://localhost:3000/api/tables/memberships/rows/1,2?pk=user_id,team_id&db=db1"
func (api *API) deleteRow(w http.ResponseWriter, r *http.Request) {
	db, ok := api.useDB(w, r)
	if !ok {
		return
	}
	table := r.PathValue("table")
	id := parsePathID(r.PathValue("id"))
	key, err := buildKey(primaryKeyColumns(r.URL.Query().Get("pk")), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := db.Delete(r.Context(), table, key); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// query executes a SELECT-style statement and returns rows.
// curl: curl -X POST -H "Content-Type: application/json" -d '{"query":"SELECT * FROM users WHERE id = ?","args":[1]}' "http://localhost:3000/api/query?db=db1"
func (api *API) query(w http.ResponseWriter, r *http.Request) {
	db, ok := api.useDB(w, r)
	if !ok {
		return
	}
	var req struct {
		Query string `json:"query"`
		Args  []any  `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rows, err := db.Query(r.Context(), req.Query, req.Args...)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

// exec executes a non-query statement and returns metadata.
// curl: curl -X POST -H "Content-Type: application/json" -d '{"query":"UPDATE users SET age = ? WHERE id = ?","args":[32,1]}' "http://localhost:3000/api/exec?db=db1"
func (api *API) exec(w http.ResponseWriter, r *http.Request) {
	db, ok := api.useDB(w, r)
	if !ok {
		return
	}
	var req struct {
		Query string `json:"query"`
		Args  []any  `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	res, err := db.Exec(r.Context(), req.Query, req.Args...)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var (
		lastInsert int64
		affected   int64
	)
	if res != nil {
		lastInsert, _ = res.LastInsertId()
		affected, _ = res.RowsAffected()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"lastInsertId": lastInsert,
		"rowsAffected": affected,
	})
}

func decodeRow(r *http.Request) (database.Row, error) {
	defer r.Body.Close()
	var row database.Row
	if err := json.NewDecoder(r.Body).Decode(&row); err != nil {
		return nil, err
	}
	return row, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func parsePathID(raw string) any {
	if raw == "" {
		return ""
	}
	if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return i
	}
	return raw
}

func queryInt(r *http.Request, key string) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return 0
	}
	i, _ := strconv.Atoi(val)
	return i
}

func (api *API) useDB(w http.ResponseWriter, r *http.Request) (database.Database, bool) {
	dbName := r.URL.Query().Get("db")
	db, err := api.connections.Get(dbName)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrConnectionMiss) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return nil, false
	}
	return db, true
}

func primaryKeyColumns(pk string) []string {
	parts := strings.Split(pk, ",")
	var cols []string
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			cols = append(cols, trimmed)
		}
	}
	if len(cols) == 0 {
		return []string{"id"}
	}
	return cols
}

func buildKey(columns []string, rawID any) (database.Key, error) {
	key := database.Key{}
	if len(columns) == 1 {
		key[columns[0]] = rawID
		return key, nil
	}

	rawStr, ok := rawID.(string)
	if !ok {
		return nil, fmt.Errorf("composite key requires comma-separated values")
	}
	values := strings.Split(rawStr, ",")
	if len(values) != len(columns) {
		return nil, fmt.Errorf("expected %d primary key values, got %d", len(columns), len(values))
	}
	for i, col := range columns {
		key[col] = parsePathID(strings.TrimSpace(values[i]))
	}
	return key, nil
}
