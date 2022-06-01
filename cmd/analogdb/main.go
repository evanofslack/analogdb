package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/evanofslack/analogdb/postgres"
	"github.com/evanofslack/analogdb/server"
)

func main() {

	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		os.Getenv("DBHOST"), os.Getenv("DBPORT"), os.Getenv("DBUSER"),
		os.Getenv("DBPASSWORD"), os.Getenv("DBNAME"))

	db := postgres.NewDB(dsn)
	if err := db.Open(); err != nil {
		log.Fatal(err)
	}

	ps := postgres.NewPostService(db)

	s := server.New()
	s.PostService = ps
	s.MountMiddleware()
	s.MountStatic()
	s.MountStatus()
	s.MountPostHandlers()
	s.Run()

}
