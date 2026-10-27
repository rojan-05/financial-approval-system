
package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Request struct {
	ID                   int       `json:"id"`
	RequestNumber        string    `json:"request_number"`
	RequestType          string    `json:"request_type"`
	RequestedBy          string    `json:"requested_by"`
	Amount               float64   `json:"amount"`
	Description          string    `json:"description"`
	Status               string    `json:"status"`
	CurrentApprovalLevel int       `json:"current_approval_level"`
	RejectionReason      string    `json:"rejection_reason"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type ApprovalHistory struct {
	ID              int       `json:"id"`
	RequestID       int       `json:"request_id"`
	ApprovalLevel   int       `json:"approval_level"`
	ApproverRole    string    `json:"approver_role"`
	ApproverName    string    `json:"approver_name"`
	Action          string    `json:"action"`
	RejectionReason string    `json:"rejection_reason"`
	CreatedAt       time.Time `json:"created_at"`
}

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Password string `json:"password,omitempty"`
}

type DashboardStats struct {
	Total          int     `json:"total"`
	Pending        int     `json:"pending"`
	Approved       int     `json:"approved"`
	Rejected       int     `json:"rejected"`
	Cancelled      int     `json:"cancelled"`
	TotalAmount    float64 `json:"total_amount"`
	PendingAmount  float64 `json:"pending_amount"`
	ApprovedAmount float64 `json:"approved_amount"`
}

type AmountReport struct {
	Tier        string  `json:"tier"`
	Label       string  `json:"label"`
	Total       int     `json:"total"`
	Approved    int     `json:"approved"`
	Rejected    int     `json:"rejected"`
	Pending     int     `json:"pending"`
	TotalAmount float64 `json:"total_amount"`
}

func main() {
	db, err := connectDB()
	if err != nil {
		log.Fatal("Database connection failed: ", err)
	}
	defer db.Close()

	log.Println("PostgreSQL connected successfully")

	if err := createTables(db); err != nil {
		log.Fatal("Table creation failed: ", err)
	}

	seedUsers(db)

	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "Financial Approval API is running",
			"status":  "success",
		})
	})

	mux.HandleFunc("/api/dashboard", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		getDashboardStats(w, r, db)
	})

	mux.HandleFunc("/api/reports/amount", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		getAmountReport(w, r, db)
	})

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getUsers(w, r, db)
		case http.MethodPost:
			createUser(w, r, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/users/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		loginUser(w, r, db)
	})

	mux.HandleFunc("/api/requests", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getRequests(w, r, db)
		case http.MethodPost:
			createRequest(w, r, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/requests/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		submitRequest(w, r, db)
	})

	mux.HandleFunc("/api/requests/decision", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		recordDecision(w, r, db)
	})

	mux.HandleFunc("/api/requests/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/requests/")
		parts := strings.Split(strings.Trim(path, "/"), "/")

		if len(parts) == 1 && parts[0] != "" {
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			getRequestDetail(w, r, db, parts[0])
		} else if len(parts) == 2 && parts[1] == "history" {
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			getApprovalHistory(w, r, db, parts[0])
		} else {
			http.NotFound(w, r)
		}
	})

	handler := corsMiddleware(mux)
	log.Println("Server running at http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", handler))
}

func createTables(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id       SERIAL PRIMARY KEY,
			name     VARCHAR(100) NOT NULL,
			role     VARCHAR(30)  NOT NULL
				CHECK (role IN ('EMPLOYEE','MANAGER','FINANCE','DIRECTOR')),
			password VARCHAR(100) NOT NULL DEFAULT '1234',
			UNIQUE (name, role)
		)`,
		`CREATE TABLE IF NOT EXISTS approval_requests (
			id                    SERIAL PRIMARY KEY,
			request_number        VARCHAR(50) UNIQUE NOT NULL,
			request_type          VARCHAR(20) NOT NULL
				CHECK (request_type IN ('EXPENSE', 'PURCHASE', 'TRAVEL')),
			requested_by          VARCHAR(100) NOT NULL,
			amount                NUMERIC(12,2) NOT NULL CHECK (amount > 0),
			description           TEXT NOT NULL,
			status                VARCHAR(30) NOT NULL DEFAULT 'DRAFT'
				CHECK (status IN ('DRAFT','PENDING_APPROVAL','APPROVED','REJECTED','CANCELLED')),
			current_approval_level INTEGER NOT NULL DEFAULT 0,
			created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS approval_history (
			id               SERIAL PRIMARY KEY,
			request_id       INTEGER NOT NULL
				REFERENCES approval_requests(id) ON DELETE CASCADE,
			approval_level   INTEGER NOT NULL,
			approver_role    VARCHAR(30) NOT NULL
				CHECK (approver_role IN ('MANAGER','FINANCE','DIRECTOR')),
			approver_name    VARCHAR(100) NOT NULL,
			action           VARCHAR(20) NOT NULL
				CHECK (action IN ('APPROVED','REJECTED')),
			rejection_reason TEXT,
			created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (request_id, approval_level)
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func seedUsers(db *sql.DB) {
	defaults := []User{
		{Name: "Manager User", Role: "MANAGER", Password: "1234"},
		{Name: "Finance User", Role: "FINANCE", Password: "1234"},
		{Name: "Director User", Role: "DIRECTOR", Password: "1234"},
	}
	for _, u := range defaults {
		_, _ = db.Exec(
			`INSERT INTO users (name, role, password) VALUES ($1, $2, $3) ON CONFLICT (name, role) DO NOTHING`,
			u.Name, u.Role, u.Password,
		)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func getDashboardStats(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var stats DashboardStats
	err := db.QueryRow(`
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status='PENDING_APPROVAL'),
			COUNT(*) FILTER (WHERE status='APPROVED'),
			COUNT(*) FILTER (WHERE status='REJECTED'),
			COUNT(*) FILTER (WHERE status='CANCELLED'),
			COALESCE(SUM(amount), 0),
			COALESCE(SUM(amount) FILTER (WHERE status='PENDING_APPROVAL'), 0),
			COALESCE(SUM(amount) FILTER (WHERE status='APPROVED'), 0)
		FROM approval_requests
	`).Scan(
		&stats.Total, &stats.Pending, &stats.Approved,
		&stats.Rejected, &stats.Cancelled,
		&stats.TotalAmount, &stats.PendingAmount, &stats.ApprovedAmount,
	)
	if err != nil {
		log.Println("Dashboard stats error:", err)
		http.Error(w, "Could not fetch stats", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func getAmountReport(w http.ResponseWriter, _ *http.Request, db *sql.DB) {
	rows, err := db.Query(`
		SELECT
			CASE
				WHEN amount < 10000 THEN 'low'
				WHEN amount BETWEEN 10000 AND 50000 THEN 'medium'
				ELSE 'high'
			END AS tier,
			COUNT(*),
			COUNT(*) FILTER (WHERE status='APPROVED'),
			COUNT(*) FILTER (WHERE status='REJECTED'),
			COUNT(*) FILTER (WHERE status='PENDING_APPROVAL'),
			COALESCE(SUM(amount),0)
		FROM approval_requests
		GROUP BY tier
		ORDER BY tier
	`)
	if err != nil {
		http.Error(w, "Could not generate report", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	labels := map[string]string{
		"low":    "Below Rs.10,000 (Manager only)",
		"medium": "Rs.10,000 to Rs.50,000 (Manager + Finance)",
		"high":   "Above Rs.50,000 (Manager + Finance + Director)",
	}

	var result []AmountReport
	for rows.Next() {
		var rep AmountReport
		if err := rows.Scan(&rep.Tier, &rep.Total, &rep.Approved, &rep.Rejected, &rep.Pending, &rep.TotalAmount); err != nil {
			continue
		}
		rep.Label = labels[rep.Tier]
		result = append(result, rep)
	}
	writeJSON(w, http.StatusOK, result)
}

func getUsers(w http.ResponseWriter, _ *http.Request, db *sql.DB) {
	rows, err := db.Query(`SELECT id, name, role FROM users ORDER BY role, name`)
	if err != nil {
		http.Error(w, "Could not fetch users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Role); err != nil {
			continue
		}
		users = append(users, u)
	}
	writeJSON(w, http.StatusOK, users)
}

func createUser(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var input struct {
		Name     string `json:"name"`
		Role     string `json:"role"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Role = strings.ToUpper(strings.TrimSpace(input.Role))
	input.Password = strings.TrimSpace(input.Password)

	if input.Name == "" || input.Password == "" {
		http.Error(w, "name and password are required", http.StatusBadRequest)
		return
	}
	validRoles := map[string]bool{"EMPLOYEE": true, "MANAGER": true, "FINANCE": true, "DIRECTOR": true}
	if !validRoles[input.Role] {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	var u User
	err := db.QueryRow(
		`INSERT INTO users (name, role, password) VALUES ($1, $2, $3) RETURNING id, name, role`,
		input.Name, input.Role, input.Password,
	).Scan(&u.ID, &u.Name, &u.Role)
	if err != nil {
		log.Println("Create user error:", err)
		http.Error(w, "Could not create user (name+role must be unique)", http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func loginUser(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var input struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Password = strings.TrimSpace(input.Password)

	var u User
	err := db.QueryRow(
		`SELECT id, name, role FROM users WHERE name=$1 AND password=$2`,
		input.Name, input.Password,
	).Scan(&u.ID, &u.Name, &u.Role)
	if err == sql.ErrNoRows {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "Login error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func getRequests(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	q := r.URL.Query()
	status := strings.ToUpper(strings.TrimSpace(q.Get("status")))
	reqType := strings.ToUpper(strings.TrimSpace(q.Get("type")))
	requestedBy := strings.TrimSpace(q.Get("requested_by"))

	query := `
		SELECT
			ar.id, ar.request_number, ar.request_type, ar.requested_by,
			ar.amount, ar.description, ar.status, ar.current_approval_level,
			COALESCE((
				SELECT ah.rejection_reason FROM approval_history ah
				WHERE ah.request_id = ar.id AND ah.action='REJECTED'
				ORDER BY ah.created_at DESC LIMIT 1
			), '') AS rejection_reason,
			ar.created_at, ar.updated_at
		FROM approval_requests ar
		WHERE 1=1
	`
	args := []any{}
	i := 1

	if status != "" {
		query += " AND ar.status = $" + strconv.Itoa(i)
		args = append(args, status)
		i++
	}
	if reqType != "" {
		query += " AND ar.request_type = $" + strconv.Itoa(i)
		args = append(args, reqType)
		i++
	}
	if requestedBy != "" {
		query += " AND ar.requested_by ILIKE $" + strconv.Itoa(i)
		args = append(args, "%"+requestedBy+"%")
		i++
	}
	query += " ORDER BY ar.created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Println("Could not fetch requests:", err)
		http.Error(w, "Could not fetch requests", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	requests := []Request{}
	for rows.Next() {
		var req Request
		if err := rows.Scan(
			&req.ID, &req.RequestNumber, &req.RequestType, &req.RequestedBy,
			&req.Amount, &req.Description, &req.Status, &req.CurrentApprovalLevel,
			&req.RejectionReason, &req.CreatedAt, &req.UpdatedAt,
		); err != nil {
			log.Println("Row scan error:", err)
			http.Error(w, "Could not read request", http.StatusInternalServerError)
			return
		}
		requests = append(requests, req)
	}
	writeJSON(w, http.StatusOK, requests)
}

func getRequestDetail(w http.ResponseWriter, _ *http.Request, db *sql.DB, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	var req Request
	err = db.QueryRow(`
		SELECT
			ar.id, ar.request_number, ar.request_type, ar.requested_by,
			ar.amount, ar.description, ar.status, ar.current_approval_level,
			COALESCE((
				SELECT ah.rejection_reason FROM approval_history ah
				WHERE ah.request_id = ar.id AND ah.action='REJECTED'
				ORDER BY ah.created_at DESC LIMIT 1
			), '') AS rejection_reason,
			ar.created_at, ar.updated_at
		FROM approval_requests ar WHERE ar.id = $1
	`, id).Scan(
		&req.ID, &req.RequestNumber, &req.RequestType, &req.RequestedBy,
		&req.Amount, &req.Description, &req.Status, &req.CurrentApprovalLevel,
		&req.RejectionReason, &req.CreatedAt, &req.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		http.Error(w, "Request not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Could not read request", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func getApprovalHistory(w http.ResponseWriter, _ *http.Request, db *sql.DB, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`
		SELECT id, request_id, approval_level, approver_role, approver_name,
		       action, COALESCE(rejection_reason,''), created_at
		FROM approval_history
		WHERE request_id = $1
		ORDER BY approval_level ASC
	`, id)
	if err != nil {
		http.Error(w, "Could not fetch history", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	history := []ApprovalHistory{}
	for rows.Next() {
		var h ApprovalHistory
		if err := rows.Scan(
			&h.ID, &h.RequestID, &h.ApprovalLevel, &h.ApproverRole,
			&h.ApproverName, &h.Action, &h.RejectionReason, &h.CreatedAt,
		); err != nil {
			continue
		}
		history = append(history, h)
	}
	writeJSON(w, http.StatusOK, history)
}

func createRequest(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var input struct {
		RequestType string  `json:"request_type"`
		RequestedBy string  `json:"requested_by"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	input.RequestType = strings.ToUpper(strings.TrimSpace(input.RequestType))
	input.RequestedBy = strings.TrimSpace(input.RequestedBy)
	input.Description = strings.TrimSpace(input.Description)

	if input.RequestedBy == "" || input.Description == "" || input.Amount <= 0 {
		http.Error(w, "Name, description and a positive amount are required", http.StatusBadRequest)
		return
	}
	if input.RequestType != "EXPENSE" && input.RequestType != "PURCHASE" && input.RequestType != "TRAVEL" {
		http.Error(w, "Invalid request type", http.StatusBadRequest)
		return
	}

	var req Request
	err := db.QueryRow(`
		INSERT INTO approval_requests
			(request_number, request_type, requested_by, amount, description)
		VALUES (
			'REQ-' || TO_CHAR(NOW(), 'YYYYMMDDHH24MISSMS') || '-' || nextval('approval_requests_id_seq'),
			$1, $2, $3, $4
		)
		RETURNING id, request_number, request_type, requested_by,
			amount, description, status, current_approval_level, created_at, updated_at
	`, input.RequestType, input.RequestedBy, input.Amount, input.Description,
	).Scan(
		&req.ID, &req.RequestNumber, &req.RequestType, &req.RequestedBy,
		&req.Amount, &req.Description, &req.Status, &req.CurrentApprovalLevel,
		&req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		log.Println("Create request failed:", err)
		http.Error(w, "Could not create request", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

func submitRequest(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var input struct {
		RequestID int `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if input.RequestID <= 0 {
		http.Error(w, "A valid request_id is required", http.StatusBadRequest)
		return
	}

	result, err := db.Exec(`
		UPDATE approval_requests
		SET status = 'PENDING_APPROVAL', current_approval_level = 1, updated_at = NOW()
		WHERE id = $1 AND status = 'DRAFT'
	`, input.RequestID)
	if err != nil {
		http.Error(w, "Could not submit request", http.StatusInternalServerError)
		return
	}
	count, err := result.RowsAffected()
	if err != nil || count == 0 {
		http.Error(w, "Draft request not found or already submitted", http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Request submitted successfully",
		"status":  "PENDING_APPROVAL",
	})
}

func recordDecision(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var input struct {
		RequestID       int    `json:"request_id"`
		ApproverName    string `json:"approver_name"`
		ApproverRole    string `json:"approver_role"`
		Action          string `json:"action"`
		RejectionReason string `json:"rejection_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	input.ApproverName = strings.TrimSpace(input.ApproverName)
	input.ApproverRole = strings.ToUpper(strings.TrimSpace(input.ApproverRole))
	input.Action = strings.ToUpper(strings.TrimSpace(input.Action))
	input.RejectionReason = strings.TrimSpace(input.RejectionReason)

	if input.RequestID <= 0 || input.ApproverName == "" {
		http.Error(w, "request_id and approver_name are required", http.StatusBadRequest)
		return
	}
	validRoles := map[string]bool{"MANAGER": true, "FINANCE": true, "DIRECTOR": true}
	if !validRoles[input.ApproverRole] {
		http.Error(w, "Invalid approver role", http.StatusBadRequest)
		return
	}
	if input.Action != "APPROVED" && input.Action != "REJECTED" {
		http.Error(w, "Action must be APPROVED or REJECTED", http.StatusBadRequest)
		return
	}
	if input.Action == "REJECTED" && input.RejectionReason == "" {
		http.Error(w, "A rejection reason is required", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Could not start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var requester string
	var amount float64
	var status string
	var level int

	err = tx.QueryRow(`
		SELECT requested_by, amount, status, current_approval_level
		FROM approval_requests WHERE id = $1 FOR UPDATE
	`, input.RequestID).Scan(&requester, &amount, &status, &level)
	if err == sql.ErrNoRows {
		http.Error(w, "Request not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Could not read request", http.StatusInternalServerError)
		return
	}

	if strings.EqualFold(requester, input.ApproverName) {
		http.Error(w, "Requesters cannot approve their own requests", http.StatusForbidden)
		return
	}
	if status != "PENDING_APPROVAL" {
		http.Error(w, "Request is not pending approval", http.StatusConflict)
		return
	}

	requiredRoles := map[int]string{1: "MANAGER", 2: "FINANCE", 3: "DIRECTOR"}
	requiredRole, ok := requiredRoles[level]
	if !ok {
		http.Error(w, "Invalid approval level", http.StatusConflict)
		return
	}
	if input.ApproverRole != requiredRole {
		http.Error(w, "Approval role does not match current level (need "+requiredRole+")", http.StatusForbidden)
		return
	}

	var alreadyActed bool
	if err = tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM approval_history WHERE request_id=$1 AND approval_level=$2
		)
	`, input.RequestID, level).Scan(&alreadyActed); err != nil {
		http.Error(w, "Could not check approval history", http.StatusInternalServerError)
		return
	}
	if alreadyActed {
		http.Error(w, "This approval level has already been processed", http.StatusConflict)
		return
	}

	if _, err = tx.Exec(`
		INSERT INTO approval_history
			(request_id, approval_level, approver_role, approver_name, action, rejection_reason)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6,''))
	`, input.RequestID, level, input.ApproverRole, input.ApproverName, input.Action, input.RejectionReason); err != nil {
		http.Error(w, "Could not save approval history", http.StatusInternalServerError)
		return
	}

	newStatus := "PENDING_APPROVAL"
	newLevel := level

	if input.Action == "REJECTED" {
		newStatus = "REJECTED"
	} else {
		maxLevel := 1
		if amount >= 10000 && amount <= 50000 {
			maxLevel = 2
		} else if amount > 50000 {
			maxLevel = 3
		}
		if level >= maxLevel {
			newStatus = "APPROVED"
		} else {
			newLevel = level + 1
		}
	}

	if _, err = tx.Exec(`
		UPDATE approval_requests
		SET status=$1, current_approval_level=$2, updated_at=NOW()
		WHERE id=$3
	`, newStatus, newLevel, input.RequestID); err != nil {
		http.Error(w, "Could not update request", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Could not commit decision", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":                "Decision recorded",
		"request_id":             input.RequestID,
		"status":                 newStatus,
		"current_approval_level": newLevel,
	})
}
