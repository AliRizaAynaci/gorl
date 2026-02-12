// Package main demonstrates using GoRL with the Echo middleware.
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/AliRizaAynaci/gorl/v2"
	"github.com/AliRizaAynaci/gorl/v2/core"
	echomw "github.com/AliRizaAynaci/gorl/v2/middleware/echo"
	"github.com/labstack/echo/v4"
)

func main() {
	limiter, err := gorl.New(core.Config{
		Strategy: core.TokenBucket,
		Limit:    5,
		Window:   30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer limiter.Close()

	e := echo.New()
	e.Use(echomw.RateLimit(limiter))

	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Hello from Echo!",
		})
	})

	log.Fatal(e.Start(":8080"))
}
