# 3rd Eye Tech Client Manager

A professional, local-first desktop application for managing clients, tracking payments, and generating financial reports. Built with Go, HTMX, and SQLite.

## 🚀 Features

- **Client Management**: Add, edit, and delete client records with detailed tracking of IP addresses, user counts, and payment frequencies.
- **Automated Billing**: Calculates total due based on user count, charges per user, and additional fees.
- **Payment Tracking**: Record receipts and maintain a history of payment cycles for every client.
- **Audit Logs**: Automatic tracking of changes to client records through "Edit History".
- **Overdue Highlighting**: Automatic visual cues (red rows) for clients who are past their due date.
- **Advanced Filtering**: Search by IP address, filter by specific due dates, and easily identify past-due accounts.
- **Excel Exports**: Generate professional styled Excel reports for any date range with automated summary totals.
- **Native Desktop Experience**: Uses Lorca to provide a sleek, chrome-based desktop interface.
- **Local-First & Secure**: Data is stored locally in a SQLite database (`ayush_clients.db`).

## 🛠️ Prerequisites

To run or build this application from source, you need:

1.  **Go (1.21 or later)**: [Download and Install Go](https://go.dev/dl/)
2.  **Google Chrome or Microsoft Edge**: Required for the Lorca desktop UI.
3.  **No CGO required**: Uses a pure Go SQLite driver, so you don't need a GCC compiler on Windows.

## 📥 Installation & Setup

1.  **Clone the repository**:
    ```bash
    git clone <repository-url>
    cd ayush
    ```

2.  **Install dependencies**:
    ```bash
    go mod download
    ```

3.  **Build the application (Windows)**:
    Use the following command to create a standalone executable that runs without a console window:
    ```bash
    go build -ldflags "-H windowsgui" -o ClientManager.exe
    ```

## 🎮 How to Use

### Starting the Software
- **From Source**: Run `go run .` in the terminal.
- **From Executable**: Double-click `ClientManager.exe`.

### Managing Clients
1.  **Add Client**: Click the "Add Client" button in the top right. Fill in the client name, IP, number of users, and payment frequency.
2.  **Edit/Delete**: Use the action buttons on the right side of the client table to update information or remove records.
3.  **Process Payment**: Click the "Receipt" button to mark a payment cycle as complete. This will automatically update the next due date based on the client's payment frequency.

### Filtering & Search
- Use the **Search Bar** to find clients by their IP address.
- Select a date and click **Filter** to see clients due on that day.
- Check **Include Past Due** to see all clients whose payment is overdue.

### Exporting Reports
1.  Enter a **Start Date** and **End Date** in the export section.
2.  Click **Export Excel**.
3.  A styled Excel file will be generated with a summary of all client dues within that period.

## 📂 Project Structure

- `main.go`: Application entry point and UI initialization.
- `db.go`: Database schema and SQLite operations.
- `handlers.go`: HTTP handlers and business logic for the frontend.
- `index.html`: The frontend UI built with HTML/CSS and HTMX.
- `ayush_clients.db`: The local database file (automatically created on first run).

## 📄 License

This software is developed for 3rd Eye Tech. All rights reserved.
