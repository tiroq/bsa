package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gopkg.in/yaml.v2"
)

type Categories map[string]float64

var userCategories = make(map[int64]Categories)

const cacheFile = "data/cache.json"

// loadCache reads cached user categories from a local file.
func loadCache() {
	file, err := os.Open(cacheFile)
	if err != nil {
		log.Printf("No cache file found, starting fresh")
		return
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&userCategories)
	if err != nil {
		log.Printf("Error decoding cache: %v", err)
	} else {
		log.Printf("Loaded categories from cache")
	}
}

// saveCache writes the current userCategories map to a local file.
func saveCache() {
	file, err := os.Create(cacheFile)
	if err != nil {
		log.Printf("Error creating cache file: %v", err)
		return
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(userCategories); err != nil {
		log.Printf("Error encoding cache: %v", err)
	}
}

// preprocessYAML inserts a space after colons if immediately followed by a digit.
func preprocessYAML(input string) string {
	re := regexp.MustCompile(`:(\d)`)
	return re.ReplaceAllString(input, ": $1")
}

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN env var required")
	}
	adminIDStr := os.Getenv("ADMIN_TELEGRAM_ID")
	if adminIDStr == "" {
		log.Fatal("ADMIN_TELEGRAM_ID env var required")
	}
	adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid ADMIN_TELEGRAM_ID: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Load previously cached categories.
	loadCache()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		userID := update.Message.From.ID
		log.Printf("Received message from %s (ID: %d): %s", update.Message.From.UserName, userID, update.Message.Text)

		// Process commands.
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				startMsg := "Welcome to Budget Splitter Assistant.\n" +
					"Send a number to split your budget based on your categories, " +
					"or upload categories in JSON or YAML format.\n" +
					"To send feedback, type: /feedback <your message>."
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, startMsg))
				log.Printf("Sent start message to user %s (ID: %d)", update.Message.From.UserName, userID)
				continue
			case "feedback":
				feedbackText := update.Message.CommandArguments()
				if strings.TrimSpace(feedbackText) == "" {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Please provide feedback text."))
					log.Printf("Feedback command without text from user %s (ID: %d)", update.Message.From.UserName, userID)
				} else {
					from := update.Message.From
					msgText := fmt.Sprintf("Feedback from %s (ID: %d):\n%s", from.UserName, from.ID, feedbackText)
					_, err := bot.Send(tgbotapi.NewMessage(adminID, msgText))
					if err != nil {
						log.Printf("Error sending feedback to admin: %v", err)
					} else {
						log.Printf("Forwarded feedback from user %s (ID: %d) to admin", from.UserName, from.ID)
					}
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Feedback sent, thank you!"))
				}
				continue
			}
		}

		text := strings.TrimSpace(update.Message.Text)

		// Attempt to parse as JSON.
		var cats Categories
		if err := json.Unmarshal([]byte(text), &cats); err == nil && len(cats) > 0 {
			userCategories[userID] = cats
			saveCache()
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Categories updated successfully via JSON."))
			log.Printf("Updated categories for user %s (ID: %d) via JSON", update.Message.From.UserName, userID)
			continue
		}

		// Attempt to parse as YAML.
		yamlText := preprocessYAML(text)
		if err := yaml.Unmarshal([]byte(yamlText), &cats); err == nil && len(cats) > 0 {
			userCategories[userID] = cats
			saveCache()
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Categories updated successfully via YAML."))
			log.Printf("Updated categories for user %s (ID: %d) via YAML", update.Message.From.UserName, userID)
			continue
		}

		// If input is a number, process the budget split.
		if amount, err := strconv.ParseFloat(text, 64); err == nil {
			cats, ok := userCategories[userID]
			if !ok || len(cats) == 0 {
				example := "No categories set. Please upload categories using JSON or YAML.\n" +
					"Example:\nJSON:\n{\"Food\": 50, \"Rent\": 30, \"Other\": 20}\n" +
					"YAML:\nFood: 50\nRent: 30\nOther: 20"
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, example))
				log.Printf("User %s (ID: %d) tried splitting budget without setting categories", update.Message.From.UserName, userID)
			} else {
				// Convert amount to integer.
				intTotal := int(math.Round(amount))
				totalWeight := 0.0
				for _, weight := range cats {
					totalWeight += weight
				}

				// Calculate each category's share as an integer value.
				portions := make(map[string]int)
				sumPortions := 0
				var firstCat string
				for cat, weight := range cats {
					raw := float64(intTotal) * (weight / totalWeight)
					intPortion := int(math.Round(raw/100) * 100)
					portions[cat] = intPortion
					sumPortions += intPortion
					if firstCat == "" {
						firstCat = cat
					}
				}

				// Adjustment: add the remaining difference to the first category.
				diff := intTotal - sumPortions
				portions[firstCat] += diff
				log.Printf("Adjustment for user %s (ID: %d): diff=%d added to %s", update.Message.From.UserName, userID, diff, firstCat)

				// Build and send the result message.
				result := "Budget split:\n"
				for cat, portion := range portions {
					result += fmt.Sprintf("%s: %d\n", cat, portion)
				}
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, result))
				log.Printf("Processed budget split for user %s (ID: %d) with int amount: %d", update.Message.From.UserName, userID, intTotal)
			}
			continue
		}

		// Fallback help message.
		help := "Send a number to split your budget based on your categories, " +
			"or upload categories in JSON or YAML format.\n" +
			"Example:\nJSON:\n{\"Food\": 50, \"Rent\": 30, \"Other\": 20}\n" +
			"YAML:\nFood: 50\nRent: 30\nOther: 20"
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, help))
		log.Printf("Sent help message to user %s (ID: %d)", update.Message.From.UserName, userID)
	}
}
