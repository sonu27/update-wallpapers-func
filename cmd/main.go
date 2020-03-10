package main

import (
	"context"
	"fmt"
	p "sonurai"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file")
	}

	ctx := context.Background()
	p.Start(ctx)
}
