
package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "hello from gin in vagrant VM",
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}
