package main

import (
	"alerting-app/config"
	"alerting-app/database"
	"alerting-app/jobs"
	"alerting-app/routes"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	app := fiber.New(fiber.Config{
		ReadBufferSize: 16 * 1024 * 1024, // 16 MB
	})

	// Configure CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "*",
		AllowHeaders:     "*",
		AllowCredentials: false,
	}))

	// Add JWT secret to every request
	app.Use(func(c *fiber.Ctx) error {
		jwtSecret := config.Config("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "your-default-secret-key" // Default fallback (consider removing in production)
		}
		c.Locals("jwtSecret", jwtSecret)
		return c.Next()
	})

	// Connect to database
	database.ConnectDB()

	// Setup routes
	routes.SetupRoutes(app)

	// Serve Vue.js frontend
	app.Static("/", "./dist")

	// Handle Vue.js SPA fallback for history mode
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./dist/index.html")
	})

	// Start cron jobs
	jobs.RunCron()

	// Start server
	log.Fatal(app.Listen(":3000"))
}
