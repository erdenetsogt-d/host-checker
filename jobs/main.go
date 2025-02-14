package jobs

import (
	"alerting-app/database"
	"alerting-app/models"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/go-co-op/gocron"
	"gorm.io/gorm"
)

var cronChecker = gocron.NewScheduler(time.UTC)

// RunCron starts the cron job to check hosts every minute
func RunCron() {
	fmt.Println("Starting cron jobs...")
	cronChecker.Every(1).Minute().Do(checkHostsInDB)
	cronChecker.StartAsync()
}

// Fetch and check hosts from the database
func checkHostsInDB() {
	db := database.DB
	fmt.Println("Checking hosts...")

	var hosts []models.Host
	if err := db.Where("is_active = ?", true).Find(&hosts).Error; err != nil {
		log.Println("Failed to retrieve active hosts:", err)
		return
	}

	now := time.Now().Add(5 * time.Second)
	var hostsToCheck []models.Host

	for _, host := range hosts {
		if shouldCheckHost(host, now) {
			hostsToCheck = append(hostsToCheck, host)
		} else {
			log.Printf("Skipping host %s as it's within the interval period.", host.Name)
		}
	}

	fmt.Println("Hosts to check:", len(hostsToCheck))

	for _, host := range hostsToCheck {
		down := checkHostStatus(&host, db)
		host.LastCheckedDate = time.Now()
		db.Save(&host)
		if down {
			handleHostDown(&host, db)
		} else {
			handleHostUp(&host, db)
		}
	}
}

// Determines if a host should be checked based on its interval
func shouldCheckHost(host models.Host, now time.Time) bool {
	interval := time.Duration(host.Interval) * time.Minute
	fmt.Printf("Checking host %s with interval %d minutes\n", host.Name, host.Interval)
	return host.LastCheckedDate.IsZero() || now.Sub(host.LastCheckedDate) > interval
}

// Checks host status based on its check method
func checkHostStatus(host *models.Host, db *gorm.DB) bool {
	var checkMethod models.CheckConfig
	if err := db.First(&checkMethod, host.MethodID).Error; err != nil {
		log.Println("Failed to fetch check method for host:", host.Name, err)
		return false
	}
	fmt.Println("Checking host", host.Name, "with method", checkMethod.Method)

	resultChan := make(chan bool)

	switch checkMethod.Method {
	case "ping":
		go func() {
			resultChan <- !pingHost(host.IP)
		}()
	case "http_get":
		go func() {
			resultChan <- !httpGetHost(host.IP, *host.ExpectedResponse)
		}()
	case "http_post":
		go func() {
			resultChan <- !httpPostHost(host.IP)
		}()
	default:
		log.Printf("Unknown check method for host %s: %s", host.Name, checkMethod.Method)
		return false
	}

	// Wait for the result from the goroutine
	return <-resultChan
}

// Runs a ping command and returns true if the host is reachable
func pingHost(ip string) bool {
	successCount := 0
	for i := 0; i < 2; i++ {
		cmd := exec.Command("ping", "-c", "1", ip)
		if err := cmd.Run(); err == nil {
			successCount++
		}
	}
	fmt.Println("Ping success count:", successCount, ip)

	return successCount > 0
}

// Performs an HTTP GET request and returns true if the host responds with 200
func httpGetHost(url string, resp_code int) bool {
	client := &http.Client{
		Timeout: 20 * time.Second, // Correct timeout value
	}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		return false
	}
	defer resp.Body.Close() // Close the response body after use

	fmt.Println("Status Code:", resp.StatusCode)
	return resp.StatusCode == resp_code
}

// Performs an HTTP POST request and returns true if the host responds with 200
func httpPostHost(url string) bool {
	resp, err := http.Post(url, "application/json", nil)
	if err != nil || resp.StatusCode != 200 {
		return false
	}
	return true
}

func handleHostDown(host *models.Host, db *gorm.DB) {
	host.IsPending = true
	host.RetryCount++
	host.LastCheckedDate = time.Now()

	fmt.Printf("Host %s is down (Retry %d)\n", host.Name, host.RetryCount)

	if host.RetryCount >= host.NumOfRetry && !host.AlertStatus {
		fmt.Printf("ALERT: %s is down!\n", host.Name)
		host.AlertStatus = true
		host.LastAlert = time.Now().Format("2006-01-02 15:04:05")
		host.IsPending = false
		writeHostHistory(db, host, "down", true)
	}
	db.Save(host)
}

func handleHostUp(host *models.Host, db *gorm.DB) {
	host.LastCheckedDate = time.Now()
	if host.AlertStatus {
		host.IsPending = false
		host.RetryCount = 0
		host.AlertStatus = false
		fmt.Printf("Host %s is back up\n", host.Name)
		writeHostHistory(db, host, "up", false)
	}

	// New change: if the host is checked and is up, reset `IsPending`
	if host.IsPending && !host.AlertStatus {
		host.IsPending = false
	}
	db.Save(host)
}
func writeHostHistory(db *gorm.DB, host *models.Host, status string, alertStatus bool) {
	history := models.HostHistory{
		HostID:      host.ID,
		HostName:    host.Name,
		Status:      status,
		CheckedAt:   time.Now(),
		AlertStatus: alertStatus,
	}
	db.Create(&history)
	sendAlert(host, alertStatus)
}

func sendAlert(host *models.Host, alertStatus bool) {
	if host.AlertChannelName == "telegram" {
		db := database.DB
		var channel models.AlertChannel

		if err := db.Where("name = ?", host.AlertChannelName).First(&channel).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				fmt.Println("Record not found")
			} else {
				fmt.Printf("Error querying the database: %v\n", err)
			}
		}
		if alertStatus {
			log.Println("Alerted for host telegram :", host.Name)
			sendTelegramAlert(host.Name+" IP is "+host.IP+" is Down :(", channel.Config1, channel.Config2, channel.Config3)

		} else {
			log.Println("Recovered for host telegram :", host.Name)
			sendTelegramAlert(host.Name+" IP is "+host.IP+" is UP :)", channel.Config1, channel.Config2, channel.Config3)

		}
	}
}

func sendTelegramAlert(message, apiurl, tkn, chat_id string) error {
	// Create the API URL
	url := fmt.Sprintf("%s%s/sendMessage", apiurl, tkn)

	// Set the parameters
	params := map[string]interface{}{
		"chat_id": chat_id,
		"text":    message,
	}

	// Marshal the parameters into JSON
	body, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %v", err)
	}

	// Send the request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set the content-type header to JSON
	req.Header.Set("Content-Type", "application/json")

	// Create an HTTP client with a timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Perform the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check for the response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
