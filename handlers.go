package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func renderClientRow(c Client) string {
	datesHtml := fmt.Sprintf("Install: %s<br>Start: %s<br>Due: %s", c.InstallDate, c.StartDate, c.DueDate)

	// Due date highlight
	rowStyle := ""
	dueStart, _ := time.Parse("2006-01-02", c.DueDate)
	today := time.Now().Truncate(24 * time.Hour)
	if !dueStart.IsZero() && !dueStart.After(today) {
		// Due date is today or passed
		rowStyle = "background-color: #fee2e2;"
	}

	return fmt.Sprintf(`<tr style="%s">
		<td>%d</td>
		<td>%s</td>
		<td>%s</td>
		<td>%d</td>
		<td>₹%.2f</td>
		<td>%d</td>
		<td>₹%.2f</td>
		<td style="font-weight:600; color:var(--success)">₹%.2f</td>
		<td>%s</td>
		<td>₹%.2f</td>
		<td>
			<button class="action-btn" hx-post="/clients/receipt?id=%d" hx-confirm="Process new receipt for %s?">Receipt</button>
			<button class="action-btn secondary" hx-get="/clients/history?id=%d" hx-target="#dialog-content" onclick="document.getElementById('edit-dialog').showModal()">History</button>
			<button class="action-btn secondary" hx-get="/clients/edithistory?id=%d" hx-target="#dialog-content" onclick="document.getElementById('edit-dialog').showModal()">Edit History</button>
		</td>
		<td>
			<button class="action-btn edit" hx-get="/clients/edit?id=%d" hx-target="#dialog-content" onclick="document.getElementById('edit-dialog').showModal()">Edit</button>
			<button class="action-btn danger" hx-delete="/clients?id=%d" hx-confirm="Are you sure you want to completely remove %s (IP: %s)?">Remove</button>
		</td>
		<td style="max-width:150px;overflow:hidden;text-overflow:ellipsis;">%s</td>
	</tr>`,
		rowStyle, c.ID, c.Name, c.IPAddress, c.Users, c.ChargesPerUser, c.PayFrequency, c.AdditionalCharges,
		c.TotalDue, datesHtml, c.LastPayment,
		c.ID, c.Name, c.ID, c.ID, c.ID, c.ID, c.Name, c.IPAddress, c.Remarks,
	)
}

func setupHandlers(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		html, err := os.ReadFile("index.html")
		if err != nil {
			http.Error(w, "Could not read index.html", http.StatusInternalServerError)
			return
		}

		today := time.Now().Format("2006-01-02")
		htmlStr := strings.Replace(string(html), "{{TODAY}}", today, -1)

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlStr))
	})

	mux.HandleFunc("/clients", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			r.ParseForm()
			users, _ := strconv.Atoi(r.FormValue("users"))
			cpu, _ := strconv.ParseFloat(r.FormValue("charges_per_user"), 64)
			freq, _ := strconv.Atoi(r.FormValue("pay_frequency"))
			addC, _ := strconv.ParseFloat(r.FormValue("additional_charges"), 64)

			startDate := r.FormValue("start_date")
			if startDate == "" {
				startDate = time.Now().Format("2006-01-02")
			}

			c := Client{
				Name:              r.FormValue("name"),
				IPAddress:         r.FormValue("ip_address"),
				Users:             users,
				ChargesPerUser:    cpu,
				PayFrequency:      freq,
				AdditionalCharges: addC,
				InstallDate:       startDate,
				StartDate:         startDate,
				DueDate:           startDate, // Due maps exactly to initial Start!
				Remarks:           r.FormValue("remarks"),
			}

			_, err := insertClient(db, c)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("HX-Trigger", "clientsChanged")
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodDelete {
			id, _ := strconv.Atoi(r.URL.Query().Get("id"))
			deleteClient(db, id)
			w.Header().Set("HX-Trigger", "clientsChanged")
			w.WriteHeader(http.StatusOK)
			return
		}
	})

	mux.HandleFunc("/clients/edit", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.URL.Query().Get("id"))

		if r.Method == http.MethodGet {
			c, err := queryClientByID(db, id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			html := fmt.Sprintf(`
				<h2>Edit Client</h2>
				<form hx-post="/clients/edit?id=%d" hx-swap="none" hx-on::after-request="document.getElementById('edit-dialog').close()">
					<div class="form-group"><label>Client Name</label><input type="text" name="name" value="%s" required></div>
					<div class="form-group"><label>IP Address</label><input type="text" name="ip_address" value="%s" required></div>
					<div class="form-row">
						<div class="form-group"><label>Users</label><input type="number" name="users" value="%d" required></div>
						<div class="form-group"><label>Charge/User (₹)</label><input type="number" step="0.01" name="charges_per_user" value="%.2f" required></div>
					</div>
					<div class="form-row">
                        <div class="form-group"><label>Pay Freq (Months)</label><input type="number" name="pay_frequency" value="%d" required></div>
                        <div class="form-group"><label>Additional Charge (₹)</label><input type="number" step="0.01" name="additional_charges" value="%.2f"></div>
                    </div>
					<div class="form-group"><label>Remarks</label><textarea name="remarks">%s</textarea></div>
					<div class="form-row">
						<div class="form-group"><label>Install Date (Uneditable)</label><input type="date" value="%s" disabled></div>
						<div class="form-group"><label>Cycle Start Date</label><input type="date" name="start_date" value="%s"></div>
						<div class="form-group"><label>Due Date</label><input type="date" name="due_date" value="%s"></div>
					</div>
					<div class="form-actions"><button type="button" class="secondary" onclick="document.getElementById('edit-dialog').close()">Cancel</button><button type="submit">Save Changes</button></div>
				</form>
			`, c.ID, c.Name, c.IPAddress, c.Users, c.ChargesPerUser, c.PayFrequency, c.AdditionalCharges, c.Remarks, c.InstallDate, c.StartDate, c.DueDate)

			w.Write([]byte(html))
			return
		}

		if r.Method == http.MethodPost {
			r.ParseForm()
			c, _ := queryClientByID(db, id)
			oldC := c

			c.Name = r.FormValue("name")
			c.IPAddress = r.FormValue("ip_address")
			c.Users, _ = strconv.Atoi(r.FormValue("users"))
			c.ChargesPerUser, _ = strconv.ParseFloat(r.FormValue("charges_per_user"), 64)
			c.PayFrequency, _ = strconv.Atoi(r.FormValue("pay_frequency"))
			c.AdditionalCharges, _ = strconv.ParseFloat(r.FormValue("additional_charges"), 64)
			c.Remarks = r.FormValue("remarks")
			if r.FormValue("start_date") != "" {
				c.StartDate = r.FormValue("start_date")
			}
			if r.FormValue("due_date") != "" {
				c.DueDate = r.FormValue("due_date")
			}

			// Record edit
			details := fmt.Sprintf("Name:%s->%s | IP:%s->%s | Users:%d->%d | Charge:%.2f->%.2f | Freq:%d->%d | Add:%.2f->%.2f | Rmks:%s->%s", oldC.Name, c.Name, oldC.IPAddress, c.IPAddress, oldC.Users, c.Users, oldC.ChargesPerUser, c.ChargesPerUser, oldC.PayFrequency, c.PayFrequency, oldC.AdditionalCharges, c.AdditionalCharges, oldC.Remarks, c.Remarks)
			insertEditHistory(db, id, details)

			updateClient(db, c)
			w.Header().Set("HX-Trigger", "clientsChanged")
			w.WriteHeader(http.StatusOK)
		}
	})

	mux.HandleFunc("/clients/receipt", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			id, _ := strconv.Atoi(r.URL.Query().Get("id"))
			c, err := queryClientByID(db, id)
			if err != nil {
				return
			}

			// New Start Date = Current Due Date
			newStart := c.DueDate

			// New Due Date = Current Due Date + Frequency(months)
			t, err := time.Parse("2006-01-02", c.DueDate)
			newDue := c.DueDate // fallback
			if err == nil {
				t = t.AddDate(0, c.PayFrequency, 0)
				newDue = t.Format("2006-01-02")
			}

			if err := processReceipt(db, id, newStart, newDue); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("HX-Trigger", "clientsChanged")
		}
	})

	mux.HandleFunc("/clients/history", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.URL.Query().Get("id"))
		recs, _ := queryReceiptHistory(db, id)

		var sb strings.Builder
		sb.WriteString("<h2>Receipt History</h2><table class='history-table'><thead><tr><th>Recorded On</th><th>Prev Start/Due</th><th>New Start/Due</th></tr></thead><tbody>")
		if len(recs) == 0 {
			sb.WriteString("<tr><td colspan='3'>No history available</td></tr>")
		} else {
			for _, rec := range recs {
				sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s to %s</td><td>%s to %s</td></tr>", rec.Recorded, rec.PrevStart, rec.PrevDue, rec.NewStart, rec.NewDue))
			}
		}
		sb.WriteString("</tbody></table><br><button type='button' class='secondary' onclick=\"document.getElementById('edit-dialog').close()\">Close</button>")
		w.Write([]byte(sb.String()))
	})

	mux.HandleFunc("/clients/edithistory", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.URL.Query().Get("id"))
		recs, _ := queryEditHistory(db, id)

		var sb strings.Builder
		sb.WriteString("<h2>Edit History</h2><table class='history-table'><thead><tr><th>Edited On</th><th>Changes Made</th></tr></thead><tbody>")
		if len(recs) == 0 {
			sb.WriteString("<tr><td colspan='2'>No edits recorded</td></tr>")
		} else {
			for _, rec := range recs {
				sb.WriteString(fmt.Sprintf("<tr><td style='white-space:nowrap'>%s</td><td style='font-size:0.8rem; word-break:break-all'>%s</td></tr>", rec.EditedAt, rec.Details))
			}
		}
		sb.WriteString("</tbody></table><br><button type='button' class='secondary' onclick=\"document.getElementById('edit-dialog').close()\">Close</button>")
		w.Write([]byte(sb.String()))
	})

	mux.HandleFunc("/clients/search", func(w http.ResponseWriter, r *http.Request) {
		searchIP := r.URL.Query().Get("searchIP")
		searchDue := r.URL.Query().Get("searchDue")
		includePast := r.URL.Query().Get("includePastDue") == "true"
		clients, err := queryClients(db, searchIP, searchDue, includePast)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var sb strings.Builder
		for _, c := range clients {
			sb.WriteString(renderClientRow(c))
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(sb.String()))
	})

	mux.HandleFunc("/export", func(w http.ResponseWriter, r *http.Request) {
		startDate := r.URL.Query().Get("startDate")
		endDate := r.URL.Query().Get("endDate")

		query := `SELECT id, name, ip_address, users, charges_per_user, pay_frequency, additional_charges, install_date, total_due, start_date, due_date, CAST(last_payment AS REAL), remarks FROM clients WHERE due_date >= ? AND due_date <= ? ORDER BY id DESC`
		rows, err := db.Query(query, startDate, endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		f := excelize.NewFile()
		sheet := "Sheet1"

		// 1. Top Header: CLIENTELE INDIA LIMITED
		f.MergeCell(sheet, "A1", "M1")
		f.SetCellValue(sheet, "A1", "CLIENTELE INDIA LIMITED")
		header1Style, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Size: 18, Family: "Calibri"},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		})
		f.SetCellStyle(sheet, "A1", "M1", header1Style)
		f.SetRowHeight(sheet, 1, 25)

		// 2. Sub Header: All Client Details As On Date <Date>
		f.MergeCell(sheet, "A2", "M2")
		f.SetCellValue(sheet, "A2", fmt.Sprintf("All Client Details As On Date %s", time.Now().Format("02-Jan-2006")))
		header2Style, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Size: 12, Family: "Calibri"},
			Alignment: &excelize.Alignment{Horizontal: "center"},
			Border: []excelize.Border{
				{Type: "bottom", Color: "000000", Style: 1},
			},
		})
		f.SetCellStyle(sheet, "A2", "M2", header2Style)

		// General Border Style for Data
		borderStyle, _ := f.NewStyle(&excelize.Style{
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 1},
				{Type: "top", Color: "000000", Style: 1},
				{Type: "bottom", Color: "000000", Style: 1},
				{Type: "right", Color: "000000", Style: 1},
			},
		})

		// 3. Table Headers (Row 3)
		headers := []string{"ID", "Name", "IP Address", "Users", "Charge/User", "Freq", "Add. Charges", "Install Date", "Total Due", "Start Date", "Due Date", "Last Payment", "Remarks"}
		headerRowStyle, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 1},
				{Type: "top", Color: "000000", Style: 1},
				{Type: "bottom", Color: "000000", Style: 1},
				{Type: "right", Color: "000000", Style: 1},
			},
		})

		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 3)
			f.SetCellValue(sheet, cell, h)
			f.SetCellStyle(sheet, cell, cell, headerRowStyle)
		}

		// Adjust column widths
		f.SetColWidth(sheet, "B", "C", 25)
		f.SetColWidth(sheet, "H", "M", 15)

		// 4. Populate Data
		rowIdx := 4
		var sumTotalDue float64

		for rows.Next() {
			var c Client
			rows.Scan(
				&c.ID, &c.Name, &c.IPAddress, &c.Users, &c.ChargesPerUser,
				&c.PayFrequency, &c.AdditionalCharges, &c.InstallDate, &c.TotalDue,
				&c.StartDate, &c.DueDate, &c.LastPayment, &c.Remarks,
			)

			sumTotalDue += c.TotalDue

			colData := []interface{}{c.ID, c.Name, c.IPAddress, c.Users, c.ChargesPerUser, c.PayFrequency, c.AdditionalCharges, c.InstallDate, c.TotalDue, c.StartDate, c.DueDate, c.LastPayment, c.Remarks}
			for i, val := range colData {
				cell, _ := excelize.CoordinatesToCellName(i+1, rowIdx)
				f.SetCellValue(sheet, cell, val)
				f.SetCellStyle(sheet, cell, cell, borderStyle)
			}
			rowIdx++
		}

		// 5. Total Row Calculation
		totalRowIdx := rowIdx
		totalLabelCell, _ := excelize.CoordinatesToCellName(1, totalRowIdx)
		totalMergeEndCell, _ := excelize.CoordinatesToCellName(8, totalRowIdx) // A to H
		totalValueCell, _ := excelize.CoordinatesToCellName(9, totalRowIdx)    // I (Total Due)
		totalEndCell, _ := excelize.CoordinatesToCellName(13, totalRowIdx)     // M

		f.MergeCell(sheet, totalLabelCell, totalMergeEndCell)
		f.SetCellValue(sheet, totalLabelCell, "Total")
		f.SetCellValue(sheet, totalValueCell, sumTotalDue)

		totalStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true},
			Alignment: &excelize.Alignment{Horizontal: "right"},
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 1},
				{Type: "top", Color: "000000", Style: 1},
				{Type: "bottom", Color: "000000", Style: 1},
				{Type: "right", Color: "000000", Style: 1},
			},
		})

		// Apply borders to the full total row
		f.SetCellStyle(sheet, totalLabelCell, totalEndCell, totalStyle)

		w.Header().Set("Content-Disposition", "attachment; filename=clients_export.xlsx")
		w.Header().Set("Content-Type", "application/octet-stream")
		f.Write(w)
	})
}
