package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

type Client struct {
	ID                int
	Name              string
	IPAddress         string
	Users             int
	ChargesPerUser    float64
	PayFrequency      int     
	AdditionalCharges float64 
	InstallDate       string 
	TotalDue          float64
	StartDate         string // Cycle Start Date
	EndDate           string // Mirror of Due Date
	DueDate           string
	LastPayment       float64
	Receipt           string // Unused now but kept for db schema
	Block             bool   // Unused now
	Remarks           string
}

type ReceiptRecord struct {
	ID        int
	ClientID  int
	PrevStart string
	PrevDue   string
	NewStart  string
	NewDue    string
	Recorded  string
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "ayush_clients.db")
	if err != nil {
		return nil, err
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS clients (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		ip_address TEXT,
		users INTEGER,
		charges_per_user REAL,
		pay_frequency INTEGER,
		additional_charges REAL,
		install_date TEXT,
		total_due REAL,
		start_date TEXT,
		end_date TEXT,
		due_date TEXT,
		last_payment TEXT,
		receipt TEXT,
		block BOOLEAN,
		remarks TEXT
	);

	CREATE TABLE IF NOT EXISTS receipt_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		client_id INTEGER,
		prev_start TEXT,
		prev_due TEXT,
		new_start TEXT,
		new_due TEXT,
		recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS edit_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		client_id INTEGER,
		details TEXT,
		edited_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(createTableQuery); err != nil {
		return nil, err
	}
	
	// Try to add install_date to existing table if it was created before this version
	db.Exec("ALTER TABLE clients ADD COLUMN install_date TEXT DEFAULT ''")
	
	return db, nil
}

type EditRecord struct {
	ID        int
	ClientID  int
	Details   string
	EditedAt  string
}

func insertEditHistory(db *sql.DB, clientID int, details string) error {
	_, err := db.Exec("INSERT INTO edit_history (client_id, details) VALUES (?, ?)", clientID, details)
	return err
}

func queryEditHistory(db *sql.DB, clientID int) ([]EditRecord, error) {
	rows, err := db.Query("SELECT id, client_id, details, datetime(edited_at, 'localtime') FROM edit_history WHERE client_id=? ORDER BY id DESC", clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []EditRecord
	for rows.Next() {
		var r EditRecord
		if err := rows.Scan(&r.ID, &r.ClientID, &r.Details, &r.EditedAt); err != nil {
			log.Println("scan error:", err)
			continue
		}
		recs = append(recs, r)
	}
	return recs, nil
}

func insertClient(db *sql.DB, c Client) (int64, error) {
	c.TotalDue = float64(c.Users)*c.ChargesPerUser*float64(c.PayFrequency) + c.AdditionalCharges
	c.LastPayment = 0 // Initially 0, populated on receipt
	
	query := `INSERT INTO clients (
		name, ip_address, users, charges_per_user, pay_frequency, 
		additional_charges, install_date, total_due, start_date, end_date, 
		due_date, last_payment, receipt, block, remarks
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	res, err := db.Exec(query,
		c.Name, c.IPAddress, c.Users, c.ChargesPerUser, c.PayFrequency,
		c.AdditionalCharges, c.InstallDate, c.TotalDue, c.StartDate, c.DueDate, // Map DueDate to EndDate
		c.DueDate, c.LastPayment, "", false, c.Remarks,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateClient(db *sql.DB, c Client) error {
	c.TotalDue = float64(c.Users)*c.ChargesPerUser*float64(c.PayFrequency) + c.AdditionalCharges
	// Preserving existing LastPayment
	
	query := `UPDATE clients SET
		name=?, ip_address=?, users=?, charges_per_user=?, pay_frequency=?, 
		additional_charges=?, start_date=?, total_due=?, end_date=?, 
		due_date=?, last_payment=?, remarks=? WHERE id=?`
	
	_, err := db.Exec(query,
		c.Name, c.IPAddress, c.Users, c.ChargesPerUser, c.PayFrequency,
		c.AdditionalCharges, c.StartDate, c.TotalDue, c.DueDate, // Mirror EndDate
		c.DueDate, c.LastPayment, c.Remarks, c.ID,
	)
	return err
}

func deleteClient(db *sql.DB, id int) error {
	_, err := db.Exec("DELETE FROM clients WHERE id=?", id)
	return err
}

func queryClientByID(db *sql.DB, id int) (Client, error) {
	var c Client
	row := db.QueryRow("SELECT id, name, ip_address, users, charges_per_user, pay_frequency, additional_charges, install_date, total_due, start_date, end_date, due_date, CAST(last_payment AS REAL), remarks FROM clients WHERE id=?", id)
	err := row.Scan(
		&c.ID, &c.Name, &c.IPAddress, &c.Users, &c.ChargesPerUser,
		&c.PayFrequency, &c.AdditionalCharges, &c.InstallDate, &c.TotalDue,
		&c.StartDate, &c.EndDate, &c.DueDate, &c.LastPayment, &c.Remarks,
	)
	if c.InstallDate == "" {
		c.InstallDate = c.StartDate // Fallback for old data
	}
	return c, err
}

func processReceipt(db *sql.DB, clientID int, newStart, newDue string) error {
	c, err := queryClientByID(db, clientID)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// 1. Insert history
	_, err = tx.Exec("INSERT INTO receipt_history (client_id, prev_start, prev_due, new_start, new_due) VALUES (?, ?, ?, ?, ?)",
		clientID, c.StartDate, c.DueDate, newStart, newDue)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. Update client and set last_payment = current TotalDue
	_, err = tx.Exec("UPDATE clients SET start_date=?, due_date=?, end_date=?, last_payment=? WHERE id=?", newStart, newDue, newDue, c.TotalDue, clientID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func queryReceiptHistory(db *sql.DB, clientID int) ([]ReceiptRecord, error) {
	rows, err := db.Query("SELECT id, client_id, prev_start, prev_due, new_start, new_due, datetime(recorded_at, 'localtime') FROM receipt_history WHERE client_id=? ORDER BY id DESC", clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []ReceiptRecord
	for rows.Next() {
		var r ReceiptRecord
		if err := rows.Scan(&r.ID, &r.ClientID, &r.PrevStart, &r.PrevDue, &r.NewStart, &r.NewDue, &r.Recorded); err != nil {
			log.Println("scan error:", err)
			continue
		}
		recs = append(recs, r)
	}
	return recs, nil
}

func queryClients(db *sql.DB, searchIP string, searchDue string, includePast bool) ([]Client, error) {
	query := `SELECT id, name, ip_address, users, charges_per_user, pay_frequency, additional_charges, install_date, total_due, start_date, end_date, due_date, CAST(last_payment AS REAL), receipt, block, remarks FROM clients WHERE ip_address LIKE ?`
	
	var args []interface{}
	args = append(args, "%"+searchIP+"%")

	if searchDue != "" {
		if includePast {
			query += " AND due_date <= ?"
		} else {
			query += " AND due_date = ?"
		}
		args = append(args, searchDue)
	}
	
	query += " ORDER BY id DESC"
	
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var c Client
		if err := rows.Scan(
			&c.ID, &c.Name, &c.IPAddress, &c.Users, &c.ChargesPerUser,
			&c.PayFrequency, &c.AdditionalCharges, &c.InstallDate, &c.TotalDue,
			&c.StartDate, &c.EndDate, &c.DueDate, &c.LastPayment,
			&c.Receipt, &c.Block, &c.Remarks,
		); err != nil {
			log.Println("scan error:", err)
			continue
		}
		if c.InstallDate == "" {
			c.InstallDate = c.StartDate // Fallback for old data
		}
		clients = append(clients, c)
	}
	return clients, nil
}
