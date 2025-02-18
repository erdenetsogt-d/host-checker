package jobs

import (
	"alerting-app/database"
	"alerting-app/models"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
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
			headers, err := parseHeaders(host.HttpHeader)
			if err != nil {
				log.Printf("Error parsing headers for host %s: %v", host.Name, err)
				resultChan <- false
				return
			}

			success := httpGetHost(host.IP, headers, getExpectedResponse(host))
			resultChan <- success
		}()
	case "http_post":
		go func() {
			var headers map[string]string
			if host.HttpHeader != nil {
				if err := json.Unmarshal([]byte(*host.HttpHeader), &headers); err != nil {
					fmt.Println("Error parsing headers:", err)
					resultChan <- false
					return
				}
			}

			var body string
			if host.HttpBody != nil {
				body = *host.HttpBody
			}

			resultChan <- !httpPostHost(host.IP, body, headers, *host.ExpectedResponse)
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
func httpGetHost(url string, headers map[string]string, resp_code int) bool {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	// Create new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return false
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return false
	}
	defer resp.Body.Close()

	// Read and print response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return false
	}

	fmt.Println("Status Code:", resp.StatusCode)
	fmt.Println("Response Body:", string(respBody))
	fmt.Println("Response Headers:", resp.Header)

	return resp.StatusCode == resp_code
}

// Performs an HTTP POST request and returns true if the host responds with 200
func httpPostHost(url, body string, headers map[string]string, resp_code int) bool {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	// Create new request
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return false
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return false
	}
	defer resp.Body.Close()

	// Read and print response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return false
	}

	fmt.Println("Status Code:", resp.StatusCode)
	fmt.Println("Response Body:", string(respBody))
	fmt.Println("Response Headers:", resp.Header)

	return resp.StatusCode == resp_code
}
func handleHostDown(host *models.Host, db *gorm.DB) {
	host.IsPending = true
	host.RetryCount++
	host.LastCheckedDate = time.Now()

	fmt.Printf("Host %s is down (Retry %d)\n", host.Name, host.RetryCount)

	// Check if the host has reached the maximum retry count and is still down
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
		// Calculate downtime in hours if the host was down before it was marked up
		if host.LastAlert != "" {
			fmt.Println("upp")
			fmt.Println(host.LastAlert)
			downDuration, err := timeSinceAlert(host.LastAlert)

			if err == nil && downDuration > 0 {
				fmt.Printf("Host %s was down for %.2f minutes\n", host.Name, downDuration)
				// Optionally, store this information in the history or log it
			}
		}
		host.IsPending = false
		host.RetryCount = 0
		host.AlertStatus = false
		host.LastNormal = time.Now().Format("2006-01-02 15:04:05")

		fmt.Printf("Host %s is back up\n", host.Name)
		writeHostHistory(db, host, "up", false)
	}

	// New change: if the host is checked and is up, reset `IsPending`
	if host.IsPending && !host.AlertStatus {
		host.IsPending = false
	}
	db.Save(host)
}

// Time since the last alert (in hours)
func timeSinceAlert(lastAlert string) (float64, error) {
	// Parse the LastAlert time
	lastAlertTime, err := time.Parse("2006-01-02 15:04:05", lastAlert)
	if err != nil {
		return 0, fmt.Errorf("failed to parse last alert time: %v", err)
	}

	// Ensure the parsed time is in UTC
	lastAlertTime = lastAlertTime.UTC()
	fmt.Println("Last Alert Time ----", lastAlertTime)

	// Get the current time in UTC
	now := time.Now().UTC().Add(8 * time.Hour)
	fmt.Println("Current Time ----", now)

	// Calculate the duration between the current time and last alert time
	duration := now.Sub(lastAlertTime).Minutes()

	// Log the duration (in minutes)
	fmt.Println("Duration (in minutes) ----", duration)

	return duration, nil
}

func writeHostHistory(db *gorm.DB, host *models.Host, status string, alertStatus bool) {
	history := models.HostHistory{
		HostID:      host.ID,
		HostName:    host.Name,
		Status:      status,
		CheckedAt:   time.Now(),
		DeviceType:  host.DeviceTypeName,
		AlertStatus: alertStatus,
	}

	// If the host was down, record the downtime in the history
	if status == "down" && host.LastAlert != "" {
		downDuration, err := timeSinceAlert(host.LastAlert)
		if err == nil && downDuration > 0 {
			history.DownDuration = downDuration // Store downtime duration in history
		}
	}

	// If the host is up, calculate the down duration and save it
	if status == "up" && host.AlertStatus {
		// Calculate downtime if the host was previously down
		if host.LastAlert != "" {
			downDuration, err := timeSinceAlert(host.LastAlert)
			if err == nil && downDuration > 0 {
				history.DownDuration = downDuration // Store downtime duration in history
			}
		}
		// Reset the host's alert status when it comes back up
		host.AlertStatus = false
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
func parseHeaders(headerStr *string) (map[string]string, error) {
	if headerStr == nil {
		return make(map[string]string), nil
	}

	headers := make(map[string]string)
	err := json.Unmarshal([]byte(*headerStr), &headers)
	if err != nil {
		return nil, fmt.Errorf("failed to parse headers: %v", err)
	}
	return headers, nil
}
func getExpectedResponse(host *models.Host) int {
	if host.ExpectedResponse == nil {
		return http.StatusOK
	}
	return *host.ExpectedResponse
}
