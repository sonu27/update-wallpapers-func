package main

import (
	"context"
	"fmt"
	"log"
	p "sonurai"

	"github.com/joho/godotenv"
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
