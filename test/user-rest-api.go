package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/user"
)

var port string

type info struct {
	Username *user.User `json:"user"`
}

func main() {
	portEnv := os.Getenv("PORT")
	if portEnv == "" {
		log.Fatal("Please Set PORT ENV VAR")
	}

	var data info
	data.Username, _ = user.Current()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response, _ := json.Marshal(data)
		w.Write(response)

	})
	log.Println("Starting Json API on port", portEnv)
	log.Fatal(http.ListenAndServe(":"+portEnv, nil))
}
