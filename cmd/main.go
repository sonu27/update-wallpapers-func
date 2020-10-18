package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	p "github.com/sonu27/update-wallpapers-func"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file")
	}

	ctx := context.Background()
	if err := p.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
