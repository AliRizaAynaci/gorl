// Package main demonstrates using GoRL with the Fiber middleware.
package main

import (
	"log"
	"time"

	"github.com/AliRizaAynaci/gorl/v2"
	"github.com/AliRizaAynaci/gorl/v2/core"
	fibermw "github.com/AliRizaAynaci/gorl/v2/middleware/fiber"
	"github.com/gofiber/fiber/v2"
)

func main() {
	limiter, err := gorl.New(core.Config{
		Strategy: core.FixedWindow,
		Limit:    5,
		Window:   30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer limiter.Close()

	app := fiber.New()
	app.Use(fibermw.RateLimit(limiter))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello from Fiber!")
	})

	log.Fatal(app.Listen(":3000"))
}
