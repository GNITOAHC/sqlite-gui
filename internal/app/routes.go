package app

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"sqlite-gui/pkg/database"
)

type API struct {
	db database.Database
}

func NewAPI(db database.Database) *API {
	return &API{db: db}
}

func (api *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tables", api.listTables)
	mux.HandleFunc("GET /api/tables/{table}/columns", api.getColumns)
	mux.HandleFunc("GET /api/tables/{table}/rows", api.getRows)
	mux.HandleFunc("POST /api/tables/{table}/rows", api.insertRow)
	mux.HandleFunc("PUT /api/tables/{table}/rows/{id}", api.updateRow)
	mux.HandleFunc("DELETE /api/tables/{table}/rows/{id}", api.deleteRow)
	mux.HandleFunc("POST /api/query", api.query)
	mux.HandleFunc("POST /api/exec", api.exec)
}

func (api *API) listTables(w http.ResponseWriter, r *http.Request) {
	tables, err := api.db.Tables(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tables": tables})
}

func (api *API) getColumns(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("table")
	cols, err := api.db.Columns(r.Context(), table)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"columns": cols})
}

func (api *API) getRows(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("table")
	limit := queryInt(r, "limit")
	offset := queryInt(r, "offset")
	rows, err := api.db.Rows(r.Context(), table, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

func (api *API) insertRow(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("table")
	row, err := decodeRow(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := api.db.Insert(r.Context(), table, row); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

func (api *API) updateRow(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("table")
	id := parsePathID(r.PathValue("id"))
	pkColumn := defaultPK(r.URL.Query().Get("pk"))
	row, err := decodeRow(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := api.db.Update(r.Context(), table, pkColumn, id, row); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (api *API) deleteRow(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("table")
	id := parsePathID(r.PathValue("id"))
	pkColumn := defaultPK(r.URL.Query().Get("pk"))
	if err := api.db.Delete(r.Context(), table, pkColumn, id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (api *API) query(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
		Args  []any  `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rows, err := api.db.Query(r.Context(), req.Query, req.Args...)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

func (api *API) exec(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
		Args  []any  `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	res, err := api.db.Exec(r.Context(), req.Query, req.Args...)
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

func defaultPK(pk string) string {
	if strings.TrimSpace(pk) == "" {
		return "id"
	}
	return pk
}
