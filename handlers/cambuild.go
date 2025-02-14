package handlers

import (
	"alerting-app/database"
	"alerting-app/models"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/gofiber/fiber/v2"
)

// Global variable to store the MediaMTX process
var mediaMTXProcess *exec.Cmd
var mu sync.Mutex // Mutex to ensure thread-safety when accessing the global process

// Configcam generates the MediaMTX configuration file based on cameras in the database
func Configcam(c *fiber.Ctx) error {
	db := database.DB
	cams := []models.Cameras{}

	// Query the database to get the list of cameras
	err := db.Find(&cams).Error
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   "Error fetching cameras from database",
			"message": err.Error(),
		})
	}

	// Start constructing the MediaMTX config string
	config := `# MediaMTX configuration file

# Core settings
rtspTransports: [tcp, udp]  # Changed from 'protocols'
rtspAddress: :8554
rtmpAddress: :1935
hlsAddress: :8888
webrtcAddress: :8889
srtAddress: :8890
api: yes
apiAddress: :9997
metrics: yes
readTimeout: 10s
writeTimeout: 10s
writeQueueSize: 512  

# Path configurations
paths:
`

	// Loop through each camera and add it to the config string
	for _, camera := range cams {
		config += fmt.Sprintf(`  %s:
    source: "%s"
    rtspTransport: %s
    sourceOnDemand: %v
`, camera.Name, camera.RTSPURL, camera.Transport, camera.OnDemand)
	}

	// Write the configuration to a YAML file
	configFilePath := "mediamtx.yml"
	err = os.WriteFile(configFilePath, []byte(config), 0644)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   "Error writing configuration to file",
			"message": err.Error(),
		})
	}

	// Return success response
	return c.JSON(fiber.Map{
		"message":     "Configuration saved successfully",
		"config_file": configFilePath,
	})
}

func GetCamera(c *fiber.Ctx) error {
	db := database.DB
	var cameras []models.Cameras
	if err := db.Limit(9).Find(&cameras).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   "Error fetching cameras from database",
			"message": err.Error(),
		})
	}
	baseURL := os.Getenv("STREAM_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8009" // Default fallback
	}
	filteredCameras := make([]map[string]interface{}, len(cameras))
	for i, cam := range cameras {
		filteredCameras[i] = map[string]interface{}{
			"id":   cam.ID,
			"name": cam.Name,
			"url":  fmt.Sprintf("%s/%s/whep", baseURL, cam.Name),
		}
	}

	return c.JSON(filteredCameras)
}
