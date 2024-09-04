package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	db "github.com/paran0iaa/TODO/DataBase"
	"github.com/paran0iaa/TODO/internal/handlers"
	"github.com/paran0iaa/TODO/internal/services"
)

func main() {
	db.CreateDb(services.GetEnv("TODO_DBFILE"))

	r := mux.NewRouter()
	r.HandleFunc("/api/nextdate", handlers.NextDateHandler).Methods("GET")
	r.HandleFunc("/api/task", handlers.CreateTask).Methods("POST")
	http.Handle("/", r)

	if err := http.ListenAndServe(":"+services.GetEnv("TODO_PORT"), handlers.WebDir()); err != nil {
		log.Fatalf("ListenAndServe: %v\n", err)
	}
}
