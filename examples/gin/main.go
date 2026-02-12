// Package main demonstrates using GoRL with the Gin middleware.
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/AliRizaAynaci/gorl/v2"
	"github.com/AliRizaAynaci/gorl/v2/core"
	ginmw "github.com/AliRizaAynaci/gorl/v2/middleware/gin"
	"github.com/gin-gonic/gin"
)

func main() {
	limiter, err := gorl.New(core.Config{
		Strategy: core.SlidingWindow,
		Limit:    5,
		Window:   30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer limiter.Close()

	r := gin.Default()
	r.Use(ginmw.RateLimit(limiter))

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello from Gin!"})
	})

	r.Run(":8080")
}
