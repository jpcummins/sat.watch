package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	username := flag.String("username", "", "Username for the new user")
	password := flag.String("password", "", "Password for the new user")
	flag.Parse()

	if *username == "" || *password == "" {
		fmt.Println("Error: username and password are required")
		flag.Usage()
		os.Exit(1)
	}

	if len(*password) < 6 {
		fmt.Println("Error: password must be at least 6 characters long")
		os.Exit(1)
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Error hashing password: %v", err)
	}

	// Get database connection from environment
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer conn.Close(context.Background())

	// Insert the user
	_, err = conn.Exec(context.Background(),
		"INSERT INTO users (username, password_hash) VALUES ($1, $2)",
		*username, string(hashedPassword))
	if err != nil {
		log.Fatalf("Error creating user: %v", err)
	}

	fmt.Printf("Successfully created user: %s\n", *username)
}
