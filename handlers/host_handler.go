package handlers

import (
	"alerting-app/database"
	"alerting-app/models"

	"github.com/gofiber/fiber/v2"
)

func CreateHost(c *fiber.Ctx) error {
	db := database.DB

	host := new(models.Host)
	if err := c.BodyParser(host); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if result := db.Create(&host); result.Error != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": result.Error.Error(),
		})
	}

	return c.Status(200).JSON(host)
}

func GetHosts(c *fiber.Ctx) error {
	db := database.DB

	var hosts []models.Host

	// Ensure soft-deleted records are included
	if result := db.Where("deleted_at IS NULL").Find(&hosts); result.Error != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": result.Error.Error(),
		})
	}

	// Return deleted hosts
	return c.JSON(hosts)
}

func GetMethod(c *fiber.Ctx) error {
	db := database.DB
	var method []models.CheckConfig
	if result := db.Find(&method); result.Error != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": result.Error.Error(),
		})
	}

	return c.Status(201).JSON(method)

}
func GetAlert(c *fiber.Ctx) error {
	db := database.DB
	var methods []models.AlertChannel
	if result := db.Find(&methods); result.Error != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": result.Error.Error(),
		})
	}

	// Create a new slice containing only ID and name
	var response []map[string]interface{}
	for _, method := range methods {
		response = append(response, map[string]interface{}{
			"ID":   method.ID,
			"name": method.Name,
		})
	}

	return c.Status(200).JSON(response)
}

func UpdateHost(c *fiber.Ctx) error {
	db := database.DB

	// Get ID from the URL
	id := c.Params("id")

	// Find the existing host
	var host models.Host
	var updateHost models.UpdatedFields
	if err := db.First(&host, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Host not found",
		})
	}

	// Parse request body
	if err := c.BodyParser(&updateHost); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	host.Name = updateHost.Name
	host.IP = updateHost.IP
	host.MethodID = updateHost.MethodID
	host.Interval = updateHost.Interval
	host.RetryCount = updateHost.RetryCount
	host.NumOfRetry = updateHost.NumOfRetry
	host.IsActive = updateHost.IsActive
	host.ExpectedResponse = updateHost.ExpectedResponse
	host.DeviceTypeName = updateHost.DevType
	// host.DeviceTypeName = updateHost.DeviceType.DevType
	// Update the existing host record
	if err := db.Save(&host).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(200).JSON(host)
}

func DeleteHost(c *fiber.Ctx) error {
	db := database.DB

	// Get ID from the URL
	id := c.Params("id")
	var host models.Host
	if err := db.First(&host, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Host not found",
		})
	}
	if err := db.Delete(&host).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(200).JSON(fiber.Map{
		"message": "Host successfully deleted (soft delete)",
	})
}

func GetHistory(c *fiber.Ctx) error {
	db := database.DB
	var history []models.HostHistory

	// Get pagination parameters from query
	page := c.QueryInt("page", 1)   // Default to page 1 if not provided
	limit := c.QueryInt("limit", 5) // Default limit to 10 if not provided
	offset := (page - 1) * limit    // Calculate offset

	// Fetch paginated history ordered by CheckedAt
	if err := db.Order("checked_at DESC").Limit(limit).Offset(offset).Find(&history).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching host history",
		})
	}

	// Count total records for pagination info
	var total int64
	db.Model(&models.HostHistory{}).Count(&total)

	// Return paginated response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":       history,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": (total + int64(limit) - 1) / int64(limit), // Calculate total pages
	})
}
