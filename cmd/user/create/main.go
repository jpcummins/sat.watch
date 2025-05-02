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
	isAdmin := flag.Bool("admin", false, "Whether the user should be an admin")
	flag.Parse()

	if *username == "" || *password == "" {
		fmt.Println("Error: username and password are required")
		flag.Usage()
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
		"INSERT INTO users (username, password_hash, is_admin) VALUES ($1, $2, $3)",
		*username, string(hashedPassword), *isAdmin)
	if err != nil {
		log.Fatalf("Error creating user: %v", err)
	}

	adminStatus := "regular"
	if *isAdmin {
		adminStatus = "admin"
	}
	fmt.Printf("Successfully created %s user: %s\n", adminStatus, *username)
}
