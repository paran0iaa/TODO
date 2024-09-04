package handlers

import (
	"net/http"

	_ "github.com/mattn/go-sqlite3"
	"github.com/paran0iaa/TODO/internal/services"
)

func WebDir() http.Handler {
	return http.FileServer(http.Dir("./web"))
}

func NextDateHandler(w http.ResponseWriter, r *http.Request) {
	now := r.URL.Query().Get("now")
	date := r.URL.Query().Get("date")
	repeat := r.URL.Query().Get("repeat")

	result, err := services.NextDate(now, date, repeat)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(result))
}

func CreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}
