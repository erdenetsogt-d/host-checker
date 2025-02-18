package handlers

import (
	"alerting-app/database"
	"alerting-app/models"

	"github.com/gofiber/fiber/v2"
)

func GetDevType(c *fiber.Ctx) error {
	db := database.DB

	var DevType []models.DeviceType
	// Query only ID and DevType, excluding soft-deleted records

	if result := db.Where("deleted_at IS NULL").Find(&DevType); result.Error != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": result.Error.Error(),
		})
	}
	return c.JSON(DevType)
}
