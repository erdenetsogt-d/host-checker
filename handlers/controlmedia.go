package handlers

import (
	"log"
	"os"
	"os/exec"

	"github.com/gofiber/fiber/v2"
)

func ControlStream(c *fiber.Ctx) error {
	mu.Lock()
	defer mu.Unlock()

	// Parse the command from the request body
	var request struct {
		Command string `json:"command"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	switch request.Command {
	case "on":
		// Start the MediaMTX stream if it's not running
		if mediaMTXProcess != nil && mediaMTXProcess.Process != nil {
			return c.JSON(fiber.Map{
				"status":  "MediaMTX is already running",
				"message": "The stream is already started.",
			})
		}

		configFilePath := "mediamtx.yml"

		// Check if the configuration file exists
		if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
			return c.Status(400).JSON(fiber.Map{
				"error":   "Configuration file not found",
				"message": "Please generate the configuration file first",
			})
		}

		// Start MediaMTX process
		cmd := exec.Command("mediamtx", configFilePath)

		// Set up pipes for stdout and stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   "Failed to create stdout pipe",
				"message": err.Error(),
			})
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   "Failed to create stderr pipe",
				"message": err.Error(),
			})
		}

		// Start the MediaMTX process
		if err := cmd.Start(); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   "Error starting MediaMTX",
				"message": err.Error(),
			})
		}

		// Store the process in the global variable
		mediaMTXProcess = cmd

		// Handle output in separate goroutines
		go func() {
			buffer := make([]byte, 1024)
			for {
				n, err := stdout.Read(buffer)
				if n > 0 {
					log.Printf("MediaMTX stdout: %s", buffer[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		go func() {
			buffer := make([]byte, 1024)
			for {
				n, err := stderr.Read(buffer)
				if n > 0 {
					log.Printf("MediaMTX stderr: %s", buffer[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		// Wait for the process in a separate goroutine
		go func() {
			err := cmd.Wait()
			if err != nil {
				log.Printf("MediaMTX process finished with error: %s", err)
			} else {
				log.Println("MediaMTX process finished successfully")
			}
			mu.Lock()
			mediaMTXProcess = nil
			mu.Unlock()
		}()

		return c.JSON(fiber.Map{
			"status":  true,
			"message": "The stream has started successfully.",
		})

	case "off":
		// Stop the MediaMTX stream if it's running
		if mediaMTXProcess == nil || mediaMTXProcess.Process == nil {
			return c.JSON(fiber.Map{
				"status":  false,
				"message": "No active stream to stop.",
			})
		}

		// Try to gracefully stop the process with SIGTERM
		if err := mediaMTXProcess.Process.Signal(os.Interrupt); err != nil {
			// Force kill if SIGTERM fails
			if killErr := mediaMTXProcess.Process.Kill(); killErr != nil {
				return c.Status(500).JSON(fiber.Map{
					"error":   "Error stopping MediaMTX",
					"message": killErr.Error(),
				})
			}
		}

		// Wait for the process to fully stop
		if err := mediaMTXProcess.Wait(); err != nil {
			log.Printf("Process exited with error: %v", err)
		}

		// Reset the process reference
		mediaMTXProcess = nil

		return c.JSON(fiber.Map{
			"status":  false,
			"message": "The stream has been stopped successfully.",
		})

	case "status":
		// Return the current status of the MediaMTX process
		if mediaMTXProcess != nil && mediaMTXProcess.Process != nil {
			return c.JSON(fiber.Map{
				"status":  true,
				"message": "MediaMTX stream is currently running.",
			})
		}

		return c.JSON(fiber.Map{
			"status":  false,
			"message": "No active stream running.",
		})

	default:
		// Invalid command
		return c.Status(400).JSON(fiber.Map{
			"error":   "Invalid command",
			"message": "Command must be 'on', 'off', or 'status'.",
		})
	}
}
