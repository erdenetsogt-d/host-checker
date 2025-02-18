package routes

import (
	"alerting-app/handlers"
	"alerting-app/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api")

	// Public routes
	api.Post("/login", handlers.Login)
	api.Get("/buildconfig", handlers.Configcam)

	api.Get("/validate-token", handlers.ValidateToken) // Optional endpoint to check token validity

	// Protected routes group
	protected := api.Group("")
	protected.Use(middleware.Protected())

	// All these routes will require authentication
	protected.Post("/hosts", handlers.CreateHost)
	// protected.Post("/controlStream", handlers.ControlStream)
	// protected.Get("/get-cameras", handlers.GetCamera)
	protected.Put("/hosts/:id", handlers.UpdateHost)
	protected.Delete("/hosts/:id", handlers.DeleteHost)
	protected.Get("/hosts", handlers.GetHosts)
	protected.Get("/devtype", handlers.GetDevType)
	protected.Get("/check-method", handlers.GetMethod)
	protected.Get("/check-alert", handlers.GetAlert)
	protected.Get("/host-history", handlers.GetHistory)
}
