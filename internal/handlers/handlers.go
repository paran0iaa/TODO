package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/paran0iaa/TODO/internal/models"
	"github.com/paran0iaa/TODO/internal/services"
)

// NextDateHandler обрабатывает HTTP-запрос для получения следующей даты задачи.
func NextDateHandler(w http.ResponseWriter, r *http.Request) {
	// Получает параметры "now", "date" и "repeat" из URL запроса.
	nowStr := r.URL.Query().Get("now")
	date := r.URL.Query().Get("date")
	repeat := r.URL.Query().Get("repeat")

	// Проверяет, что все необходимые параметры присутствуют.
	if nowStr == "" || date == "" || repeat == "" {
		http.Error(w, "Отсутствуют необходимые параметры", http.StatusBadRequest)
		return
	}

	// Преобразует параметр "now" в формат времени.
	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		// Если формат некорректен, возвращает ошибку клиенту.
		http.Error(w, "Некорректный формат времени", http.StatusBadRequest)
		return
	}

	// Вызывает функцию для вычисления следующей даты.
	nextDate, err := services.NextDate(now, date, repeat)
	if err != nil {
		// Если возникает ошибка, возвращает ее клиенту.
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Возвращаем только следующую дату
	fmt.Fprintln(w, nextDate)
}

func MarkTaskDone(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		services.SendJSONError(w, "Не указан идентификатор задачи", http.StatusBadRequest)
		return
	}

	db, err := sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		services.SendJSONError(w, "Ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var task models.Task
	err = db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			services.SendJSONError(w, "Задача не найдена", http.StatusNotFound)
		} else {
			services.SendJSONError(w, "Ошибка при получении задачи: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if task.Repeat == "" {
		// Одноразовая задача, удаляем ее
		_, err = db.Exec("DELETE FROM scheduler WHERE id = ?", id)
		if err != nil {
			services.SendJSONError(w, "Ошибка при удалении задачи: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Периодическая задача, рассчитываем следующую дату
		// Используем дату задачи для вычисления следующей даты
		parsedDate, err := time.Parse("20060102", task.Date)
		if err != nil {
			services.SendJSONError(w, "Некорректный формат даты задачи", http.StatusInternalServerError)
			return
		}

		nextDate, err := services.NextDate(parsedDate, task.Date, task.Repeat)
		if err != nil {
			services.SendJSONError(w, "Ошибка при вычислении следующей даты: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Обновляем дату задачи на следующую
		_, err = db.Exec("UPDATE scheduler SET date = ? WHERE id = ?", nextDate, id)
		if err != nil {
			services.SendJSONError(w, "Ошибка при обновлении задачи: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Возвращаем пустой JSON при успешном завершении
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(map[string]string{})
}

// tasksHandler обрабатывает GET-запросы к /api/tasks
func TasksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		services.SendJSONError(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	db, err := sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		services.SendJSONError(w, "Ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Устанавливаем значение лимита по умолчанию
	limit := 20 // Рекомендуемое количество задач

	// Получаем параметр limit из запроса, если он указан
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	// Получаем параметры search и date из запроса
	search := r.URL.Query().Get("search")
	dateParam := r.URL.Query().Get("date")

	var rows *sql.Rows

	// Построение SQL-запроса на основе параметров
	if search != "" {
		// Поиск по подстроке в полях title и comment
		searchPattern := "%" + search + "%"
		query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE ? OR comment LIKE ? ORDER BY date ASC LIMIT ?"
		rows, err = db.Query(query, searchPattern, searchPattern, limit)
	} else if dateParam != "" {
		// Фильтрация по дате
		date := dateParam
		if len(dateParam) == 10 && strings.Contains(dateParam, ".") {
			// Преобразование даты из формата DD.MM.YYYY в YYYYMMDD
			t, err := time.Parse("02.01.2006", dateParam)
			if err != nil {
				services.SendJSONError(w, "Некорректный формат даты, ожидается YYYYMMDD или DD.MM.YYYY", http.StatusBadRequest)
				return
			}
			date = t.Format("20060102")
		}

		query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE date = ? ORDER BY date ASC LIMIT ?"
		rows, err = db.Query(query, date, limit)
	} else {
		// Получение всех задач, отсортированных по дате
		query := "SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date ASC LIMIT ?"
		rows, err = db.Query(query, limit)
	}

	if err != nil {
		services.SendJSONError(w, "Ошибка при получении задач: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Инициализируем слайс задач как пустой, чтобы не получить null в JSON
	tasks := make([]models.Task, 0)

	for rows.Next() {
		var task models.Task
		var id int64
		if err := rows.Scan(&id, &task.Date, &task.Title, &task.Comment, &task.Repeat); err != nil {
			services.SendJSONError(w, "Ошибка при чтении задачи: "+err.Error(), http.StatusInternalServerError)
			return
		}
		task.ID = strconv.FormatInt(id, 10)
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		services.SendJSONError(w, "Ошибка при обработке задач: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем задачи в формате JSON
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks})
}

// taskHandler обрабатывает маршруты для /api/task
func TaskHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		services.AddTask(w, r)
	case http.MethodGet:
		if id := r.URL.Query().Get("id"); id != "" {
			services.GetTaskByID(w, r, id)
		} else {
			// Возвращаем ошибку при отсутствии идентификатора
			services.SendJSONError(w, "Не указан идентификатор", http.StatusBadRequest)
		}
	case http.MethodPut:
		services.UpdateTask(w, r)
	case http.MethodDelete:
		// Обработчик для удаления задачи
		id := r.URL.Query().Get("id")
		if id == "" {
			services.SendJSONError(w, "Не указан идентификатор", http.StatusBadRequest)
			return
		}

		db, err := sql.Open("sqlite3", "./scheduler.db")
		if err != nil {
			services.SendJSONError(w, "Ошибка сервера: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer db.Close()

		res, err := db.Exec("DELETE FROM scheduler WHERE id = ?", id)
		if err != nil {
			services.SendJSONError(w, "Ошибка при удалении задачи: "+err.Error(), http.StatusInternalServerError)
			return
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			services.SendJSONError(w, "Ошибка при проверке удаления задачи: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if rowsAffected == 0 {
			services.SendJSONError(w, "Задача не найдена", http.StatusNotFound)
			return
		}

		// Возвращаем пустой JSON при успешном удалении
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		json.NewEncoder(w).Encode(map[string]string{})

	default:
		services.SendJSONError(w, "Метод не разрешен", http.StatusMethodNotAllowed)
	}
}
