package models

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/lib/pq"
)

var db *sql.DB

func InitDB(prod bool) error {
	var psqlInfo string
	if prod {
		conn, _ := pq.ParseURL(os.Getenv("DATABASE_URL"))
		psqlInfo = conn + "sslmode=require"
	} else {
		LoadEnv()
		psqlInfo = fmt.Sprintf("host=%s port=%s user=%s "+
			"password=%s dbname=%s sslmode=disable",
			os.Getenv("DBHOST"), os.Getenv("DBPORT"), os.Getenv("DBUSER"),
			os.Getenv("DBPASSWORD"), os.Getenv("DBNAME"))
	}

	var err error

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return err
	}
	return db.Ping()
}

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
}