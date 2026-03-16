package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	_ "modernc.org/sqlite"
)

// schema is the database schema.
// All business logic lives here — not in Go.
const schema = `
CREATE TABLE IF NOT EXISTS accounts (
	id   INTEGER PRIMARY KEY,
	name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS transactions (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL REFERENCES accounts(id),
	amount     INTEGER NOT NULL, -- positive = credit, negative = debit
	note       TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Business rule: balance is always derived, never stored.
CREATE VIEW IF NOT EXISTS account_balances AS
	SELECT a.id, a.name, COALESCE(SUM(t.amount), 0) AS balance
	FROM accounts a
	LEFT JOIN transactions t ON t.account_id = a.id
	GROUP BY a.id;

-- Business rule: no overdrafts — enforced by the database, not the app.
CREATE TRIGGER IF NOT EXISTS no_overdraft
BEFORE INSERT ON transactions
WHEN NEW.amount < 0
BEGIN
	SELECT RAISE(ABORT, 'insufficient funds')
	WHERE (
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE account_id = NEW.account_id
	) + NEW.amount < 0;
END;
`

func seed(db *sql.DB) {
	db.Exec(`INSERT OR IGNORE INTO accounts (id, name) VALUES (1, 'Alice'), (2, 'Bob')`)
	db.Exec(`INSERT OR IGNORE INTO transactions (account_id, amount, note) VALUES (1, 1000, 'initial deposit'), (2, 500, 'initial deposit')`)
}

// GET /accounts — the view does all the work.
func listAccounts(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), `SELECT id, name, balance FROM account_balances`)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		type Account struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Balance int    `json:"balance"`
		}
		var accounts []Account
		for rows.Next() {
			var a Account
			rows.Scan(&a.ID, &a.Name, &a.Balance)
			accounts = append(accounts, a)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(accounts)
	}
}

// POST /transfer — the trigger enforces the overdraft rule.
func transfer(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FromID int `json:"from_id"`
			ToID   int `json:"to_id"`
			Amount int `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", 400)
			return
		}

		tx, err := db.BeginTx(r.Context(), nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer tx.Rollback()

		if _, err = tx.Exec(`INSERT INTO transactions (account_id, amount, note) VALUES (?, ?, 'transfer out')`, req.FromID, -req.Amount); err != nil {
			http.Error(w, err.Error(), 400) // trigger fires here
			return
		}
		if _, err = tx.Exec(`INSERT INTO transactions (account_id, amount, note) VALUES (?, ?, 'transfer in')`, req.ToID, req.Amount); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		tx.Commit()
		w.WriteHeader(http.StatusNoContent)
	}
}

func main() {
	db, err := sql.Open("sqlite", "bank.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err = db.Exec(schema); err != nil {
		log.Fatal(err)
	}
	seed(db)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts", listAccounts(db))
	mux.HandleFunc("POST /transfer", transfer(db))

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
