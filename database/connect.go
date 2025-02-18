package database

import (
	"alerting-app/config"
	"alerting-app/models"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ConnectDB connect to db
func ConnectDB() {
	// Retrieve environment variables
	dbHost := config.Config("DB_HOST")
	dbPort := config.Config("DB_PORT")
	dbUser := config.Config("DB_USER")
	dbPassword := config.Config("DB_PASSWORD")
	dbName := config.Config("DB_NAME")

	// Construct the connection string
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		os.Exit(1)
	}

	fmt.Println("Starting AutoMigrate...")
	err = db.AutoMigrate(
		&models.CheckConfig{},
		&models.HostHistory{},
		&models.AlertChannel{},
		&models.Host{},
		&models.SendTxt{},
		&models.User{},
		&models.Cameras{},
		&models.DeviceType{},
	)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	fmt.Println("AutoMigrate completed successfully!")

	DB = db

	// Create default admin user after successful migration
	createDefaultUser()
	createMethods()
}

func createDefaultUser() {
	var user models.User

	// Check if default user exists
	result := DB.Where("username = ?", "admin").First(&user)
	if result.Error == gorm.ErrRecordNotFound {
		// Default credentials - consider moving to environment variables
		defaultUsername := config.Config("DEFAULT_ADMIN_USERNAME")
		defaultPassword := config.Config("DEFAULT_ADMIN_PASSWORD")

		// If not set in environment, use fallback defaults
		if defaultUsername == "" {
			defaultUsername = "admin"
		}
		if defaultPassword == "" {
			defaultPassword = "admin123"
		}

		// Create password hash
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			return
		}

		// Create default user
		defaultUser := models.User{
			Username: defaultUsername,
			Password: string(hashedPassword),
		}

		result := DB.Create(&defaultUser)
		if result.Error != nil {
			log.Printf("Error creating default user: %v", result.Error)
			return
		}

		log.Printf("Default admin user '%s' created successfully", defaultUsername)
	} else {
		log.Println("Default admin user already exists")
	}
}

func createMethods() {
	var methods []models.CheckConfig
	var alertchannels []models.AlertChannel
	var devtype []models.DeviceType
	// Check if methods already exist in the table
	result := DB.Find(&methods)
	DB.Find(&devtype)
	DB.Find(&alertchannels)
	if result.Error != nil {
		log.Printf("Error checking methods: %v", result.Error)
		return
	}

	// If the table is empty, insert the default values
	if len(methods) == 0 {
		defaultMethods := []models.CheckConfig{
			{ID: 1, Method: "http_post"},
			{ID: 2, Method: "http_get"},
			{ID: 3, Method: "ping"},
		}

		for _, method := range defaultMethods {
			result := DB.Create(&method)
			if result.Error != nil {
				log.Printf("Error creating method '%s': %v", method.Method, result.Error)
			} else {
				log.Printf("Method '%s' created successfully", method.Method)
			}
		}
	} else {
		log.Println("Methods already exist in the table")
	}
	if len(alertchannels) == 0 {
		defaultChannels := []models.AlertChannel{
			{Name: "telegram"},
			{Name: "mail"},
			{Name: "hook"},
		}
		for _, channel := range defaultChannels {
			result := DB.Create(&channel)
			if result.Error != nil {
				log.Printf("Error creating method '%s': %v", channel.Name, result.Error)
			} else {
				log.Printf("Method '%s' created successfully", channel.Name)
			}

		}

	} else {
		log.Println("Channels already exist in the table")

	}

	if len(devtype) == 0 {
		defaultTypes := []models.DeviceType{
			{DevType: "Switch"},
			{DevType: "Camera"},
			{DevType: "Host"},
			{DevType: "Server"},
			{DevType: "API"},
		}
		for _, types := range defaultTypes {
			result := DB.Create(&types)
			if result.Error != nil {
				log.Printf("Error creating method '%s': %v", types.DevType, result.Error)
			} else {
				log.Printf("Method '%s' created successfully", types.DevType)
			}

		}

	} else {
		log.Println("Devtype already exist in the table")

	}

}
