package dcrpg

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq" // Start the PostgreSQL driver
)

func Connect(host, port, user, pass, dbname string) (*sql.DB, error) {
	var psqlInfo string
	if pass == "" {
		psqlInfo = fmt.Sprintf("host=%s user=%s "+
			"dbname=%s sslmode=disable",
			host, user, dbname)
	} else {
		psqlInfo = fmt.Sprintf("host=%s ser=%s "+
			"password=%s dbname=%s sslmode=disable",
			host, user, pass, dbname)
	}

	if !strings.HasPrefix(host, "/") {
		psqlInfo += fmt.Sprintf(" port=%s", port)
	}

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}
	//defer db.Close()

	err = db.Ping()
	return db, err
}
