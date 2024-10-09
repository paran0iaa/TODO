package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/paran0iaa/TODO/internal/models"
)

func WebDir() http.Handler {
	return http.FileServer(http.Dir("./web"))
}

// SendJSONError отправляет ошибку в формате JSON
func SendJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// CreateDB инициализирует базу данных SQLite и создает таблицу.
func CreateDB() {
	db, err := sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		log.Fatalf("Ошибка при открытии базы данных: %v", err)
	}
	defer db.Close()

	commands := []string{
		`CREATE TABLE IF NOT EXISTS scheduler (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date CHAR(8),
			title TEXT,
			comment TEXT,
			repeat CHAR (128)
					)`,
		"CREATE INDEX IF NOT EXISTS indexdate ON scheduler (date)",
	}

	for _, cmd := range commands {
		if _, err := db.Exec(cmd); err != nil {
			log.Fatalf("Ошибка при выполнении команды: %s, ошибка: %v", cmd, err)
		}
	}
}

func NextDate(now time.Time, date string, repeat string) (string, error) {
	// Нормализуем 'now' до полуночи
	nowDateStr := now.Format("20060102")
	now, err := time.Parse("20060102", nowDateStr)
	if err != nil {
		return "", err
	}

	if repeat == "" {
		return "", errors.New("правило повторения отсутствует")
	}

	rep := strings.Split(repeat, " ")

	if len(rep) < 1 {
		return "", errors.New("некорректное правило повторения")
	}

	timBase, err := time.Parse("20060102", date)
	if err != nil {
		return "", err
	}

	if rep[0] == "y" {
		// Извлекаем день и месяц исходной даты
		origDay := timBase.Day()
		origMonth := timBase.Month()

		for {
			// Прибавляем один год
			timBase = timBase.AddDate(1, 0, 0)

			// Проверяем, совпадают ли месяц и день
			if timBase.Day() == origDay && timBase.Month() == origMonth {
				// Проверяем, что дата после текущей
				if timBase.After(now) {
					break
				}
			} else {
				// Если дата изменилась из-за високосного года, устанавливаем на 1 марта
				timBase = time.Date(timBase.Year(), time.March, 1, 0, 0, 0, 0, timBase.Location())
				if timBase.After(now) {
					break
				}
			}
		}
		return timBase.Format("20060102"), nil
	}

	if rep[0] == "d" {
		if len(rep) < 2 {
			return "", errors.New("некорректно указан режим повторения")
		}

		days, err := strconv.Atoi(rep[1])
		if err != nil {
			return "", err // Возвращаем ошибку, если количество дней некорректно
		}

		if days > 400 {
			return "", errors.New("перенос события более чем на 400 дней недопустим")
		}

		// Добавляем дни до тех пор, пока дата не станет после текущей
		for {
			timBase = timBase.AddDate(0, 0, days)
			if timBase.After(now) {
				break
			}
		}
		return timBase.Format("20060102"), nil
	}

	return "", errors.New("некорректное правило повторения")
}

// AddTask добавляет новую задачу в базу данных
func AddTask(w http.ResponseWriter, r *http.Request) {
	var task models.Task

	// Декодирование JSON-запроса
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		SendJSONError(w, "Ошибка десериализации JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Проверка обязательного поля Title
	if task.Title == "" {
		SendJSONError(w, "Не указан заголовок задачи", http.StatusBadRequest)
		return
	}

	// Установка текущей даты, если поле date не указано
	now := time.Now()
	nowDateStr := now.Format("20060102")
	now, _ = time.Parse("20060102", nowDateStr)

	if task.Date == "" {
		task.Date = nowDateStr
	}

	parsedDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		SendJSONError(w, "Дата представлена в неправильном формате, ожидается YYYYMMDD", http.StatusBadRequest)
		return
	}

	if task.Repeat != "" {
		// Если задача повторяющаяся и дата в прошлом, вычисляем следующую дату
		if parsedDate.Before(now) {
			nextDate, err := NextDate(now, task.Date, task.Repeat)
			if err != nil {
				SendJSONError(w, "Правило повторения указано в неправильном формате: "+err.Error(), http.StatusBadRequest)
				return
			}
			task.Date = nextDate
		}
	} else {
		// Если задача не повторяющаяся и дата в прошлом, устанавливаем сегодняшнюю дату
		if parsedDate.Before(now) {
			task.Date = nowDateStr
		}
	}

	// Открытие базы данных
	db, err := sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		SendJSONError(w, "Ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Вставка новой задачи в базу данных
	stmt, err := db.Prepare("INSERT INTO scheduler(date, title, comment, repeat) VALUES (?, ?, ?, ?)")
	if err != nil {
		SendJSONError(w, "Ошибка при подготовке запроса: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	res, err := stmt.Exec(task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		SendJSONError(w, "Ошибка при вставке задачи: "+err.Error(), http.StatusInternalServerError)
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		SendJSONError(w, "Ошибка при получении ID задачи: "+err.Error(), http.StatusInternalServerError)
		return
	}

	task.ID = strconv.FormatInt(id, 10)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(map[string]string{"id": task.ID})
}

// getTasks извлекает задачи из базы данных и возвращает их в формате JSON.
func GetTasks(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		http.Error(w, "Ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date ASC")
	if err != nil {
		http.Error(w, "Ошибка при получении задач: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []map[string]string
	for rows.Next() {
		var id, date, title, comment, repeat string
		if err := rows.Scan(&id, &date, &title, &comment, &repeat); err != nil {
			http.Error(w, "Ошибка при сканировании задачи: "+err.Error(), http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, map[string]string{
			"id":      id,
			"date":    date,
			"title":   title,
			"comment": comment,
			"repeat":  repeat,
		})
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Ошибка при чтении данных задач: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks})
}

// getTaskByID извлекает задачу по идентификатору из базы данных и возвращает ее в формате JSON.
func GetTaskByID(w http.ResponseWriter, r *http.Request, id string) {
	db, err := sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		http.Error(w, "Ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var task struct {
		ID      string `json:"id"`
		Date    string `json:"date"`
		Title   string `json:"title"`
		Comment string `json:"comment"`
		Repeat  string `json:"repeat"`
	}

	err = db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			json.NewEncoder(w).Encode(map[string]string{"error": "Задача не найдена"})
		} else {
			http.Error(w, "Ошибка при получении задачи: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(task)
}

// updateTask обновляет существующую задачу в базе данных
func UpdateTask(w http.ResponseWriter, r *http.Request) {
	var task models.Task

	// Декодирование JSON-запроса
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		SendJSONError(w, "Ошибка десериализации JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Проверка обязательного поля ID
	if task.ID == "" {
		SendJSONError(w, "Не указан идентификатор задачи", http.StatusBadRequest)
		return
	}

	// Проверка обязательного поля Title
	if task.Title == "" {
		SendJSONError(w, "Не указан заголовок задачи", http.StatusBadRequest)
		return
	}

	// Установка текущей даты, если поле date не указано
	now := time.Now()
	nowDateStr := now.Format("20060102")
	now, _ = time.Parse("20060102", nowDateStr)

	if task.Date == "" {
		task.Date = nowDateStr
	}

	parsedDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		SendJSONError(w, "Дата представлена в неправильном формате, ожидается YYYYMMDD", http.StatusBadRequest)
		return
	}

	if task.Repeat != "" {
		// Если задача повторяющаяся и дата в прошлом, вычисляем следующую дату
		if parsedDate.Before(now) {
			nextDate, err := NextDate(now, task.Date, task.Repeat)
			if err != nil {
				SendJSONError(w, "Ошибка в правиле повторения: "+err.Error(), http.StatusBadRequest)
				return
			}
			task.Date = nextDate
		}
	} else {
		// Если задача не повторяющаяся и дата в прошлом, устанавливаем сегодняшнюю дату
		if parsedDate.Before(now) {
			task.Date = nowDateStr
		}
	}

	// Открытие базы данных
	db, err := sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		SendJSONError(w, "Ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Проверяем, существует ли задача с таким ID
	var existingID string
	err = db.QueryRow("SELECT id FROM scheduler WHERE id = ?", task.ID).Scan(&existingID)
	if err != nil {
		if err == sql.ErrNoRows {
			SendJSONError(w, "Задача не найдена", http.StatusNotFound)
		} else {
			SendJSONError(w, "Ошибка при проверке задачи: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Обновление задачи в базе данных
	stmt, err := db.Prepare("UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?")
	if err != nil {
		SendJSONError(w, "Ошибка при подготовке запроса: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(task.Date, task.Title, task.Comment, task.Repeat, task.ID)
	if err != nil {
		SendJSONError(w, "Ошибка при обновлении задачи: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем пустой JSON при успешном обновлении
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(map[string]string{})
}
