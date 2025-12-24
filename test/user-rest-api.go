package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"

	_ "github.com/lib/pq"
)

var port string

type info struct {
	Username *user.User `json:"user"`
	DBStatus string     `json:"db_status"`
	DBError  string     `json:"db_error,omitempty"`
}

func main() {
	portEnv := os.Getenv("PORT")
	if portEnv == "" {
		log.Fatal("Please Set PORT ENV VAR")
	}

	// Connect to DB using Peer Auth (user=current_os_user dbname=current_os_user)
	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	// We try to connect. We don't specify host, relying on default unix socket search paths.
	connStr := fmt.Sprintf("user=%s dbname=%s sslmode=disable", u.Username, u.Username)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Failed to open DB driver: %v", err)
	}
	// We don't defer close here because we want to keep it open for the handler, 
	// or we can Open/Close per request. For this simple app, keeping it open is fine 
	// but we should handle the error if Open fails (it usually doesn't connect immediately).

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var data info
		data.Username, _ = user.Current()

		if db == nil {
			data.DBStatus = "Driver Error"
		} else if err := db.Ping(); err != nil {
			data.DBStatus = "Failed"
			data.DBError = err.Error()
		} else {
			data.DBStatus = "Connected"
		}

		response, _ := json.Marshal(data)
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	})
	
	log.Println("Starting Json API on port", portEnv)
	log.Fatal(http.ListenAndServe(":"+portEnv, nil))
}