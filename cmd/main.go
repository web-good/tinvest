package main

import (
	"context"
	"log"
	"tinvest/internal/app"
)

func main() {
	ctx := context.Background()
	a, err := app.InitApp(ctx)

	if err != nil {
		log.Fatalf("failed to init app: %s", err.Error())
	}

	err = a.Run(ctx)

	if err != nil {
		log.Fatalf("failed to run app: %s", err.Error())
	}
}
