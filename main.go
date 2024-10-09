package main

import (
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/paran0iaa/TODO/internal/handlers"
	"github.com/paran0iaa/TODO/internal/services"
)

func main() {
	if _, err := os.Stat("./scheduler.db"); os.IsNotExist(err) {
		services.CreateDB()
	}

	http.HandleFunc("/api/task", handlers.TaskHandler)
	http.HandleFunc("/api/tasks", handlers.TasksHandler)
	http.HandleFunc("/api/task/done", handlers.MarkTaskDone)
	http.Handle("/", http.FileServer(http.Dir("./web")))
	http.HandleFunc("/api/nextdate", handlers.NextDateHandler)
	log.Fatal(http.ListenAndServe(":1818", nil))
}
