package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/cznic/ql"
	"github.com/erikdubbelboer/golangmeetup-chat/message"
	"github.com/julienschmidt/httprouter"
)

var (
	db *sql.DB
)

func indexHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	f, err := os.Open("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := io.Copy(w, f); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func messagesHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	since, _ := strconv.ParseInt(values.Get("since"), 10, 64)

	if err := getMessages(w, since); err != nil {
		if myerr, ok := err.(MyError); ok {
			// do something with myerr.query
			_ = myerr
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type MyError struct {
	err   error
	query string
}

func (err MyError) Error() string {
	return fmt.Sprintf("Error %v in query %s", err.err, err.query)
}

func getMessages(w http.ResponseWriter, since int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		SELECT
			id,
			from_name,
			body
		FROM messages
		WHERE id > $1
		ORDER BY id ASC
		LIMIT 10
	`

	rows, err := tx.Query(query, since)
	if err != nil {
		return MyError{
			err:   err,
			query: query,
		}
	}

	messages := make([]message.Message, 0)

	for rows.Next() {
		var m message.Message

		if err := rows.Scan(
			&m.ID,
			&m.FromName,
			&m.Body,
		); err != nil {
			return err
		}

		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(messages); err != nil {
		return err
	}

	return nil
}

func newMessageHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var m message.Message
	if err := json.NewDecoder(req.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Body.Close()

	m.SetNextID()

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	databaseHadError := false

	defer func() {
		if databaseHadError {
			if err := tx.Rollback(); err != nil {
				log.Printf("[ERR] %v", err)
			}
		} else {
			if err = tx.Commit(); err != nil {
				log.Printf("[ERR] %v", err)
			}
		}
	}()

	if _, err := tx.Exec(`
		INSERT INTO messages
		VALUES($1, $2, $3)
	`, m.ID, m.FromName, m.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		databaseHadError = true
		return
	}
}

func setup() error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	databaseHadError := false

	defer func() {
		if databaseHadError {
			if err := tx.Rollback(); err != nil {
				log.Printf("[ERR] %v", err)
			}
		} else {
			if err = tx.Commit(); err != nil {
				log.Printf("[ERR] %v", err)
			}
		}
	}()

	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id INT,
			from_name STRING,
			body STRING
		)
	`); err != nil {
		databaseHadError = true
		return err
	}

	row := tx.QueryRow(`
		SELECT id
		FROM messages
		ORDER BY id DESC
		LIMIT 1
	`)

	var id int
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			id = 1
		} else {
			databaseHadError = true
			return err
		}
	}
	message.SetNextMessageID(id)

	return nil
}

func main() {
	ql.RegisterDriver()

	if ndb, err := sql.Open("ql", "./db.db"); err != nil {
		panic(err)
	} else {
		db = ndb
		defer db.Close()
	}

	if err := setup(); err != nil {
		panic(err)
	}

	router := httprouter.New()
	router.GET("/", indexHandler)
	router.GET("/messages", messagesHandler)
	router.POST("/newmessage", newMessageHandler)

	log.Printf("[INFO] listening now...")

	if err := http.ListenAndServe("0.0.0.0:9090", router); err != nil {
		panic(err)
	}
}
