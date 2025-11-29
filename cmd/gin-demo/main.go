package main

import (
	"os"

	"github.com/nexusriot/gin-vagrant-demo/internal/server"
)

func main() {
	r := server.NewRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}
