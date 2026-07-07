//go:build linux

package main

import (
	"context"
	"fmt"
	"os"

	"snailproxy/internal/app"
	"snailproxy/internal/features"
)

func main() {
	if err := app.Run(context.Background(), os.Args[1:], features.Default()); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
