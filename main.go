package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	  // Загружаем переменные окружения из файла .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// Получаем токен из переменной окружения
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is not set")
	}

	// Инициализируем бота
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	bot.Debug = true // Включаем отладку (опционально)

	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Открываем соединение с базой данных
	db, err := sql.Open("sqlite3", "./auto_parts.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Создаем таблицы, если они не существуют
	createTables(db)

	// Создаем канал для получения обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Обрабатываем входящие сообщения
	for update := range updates {
		if update.Message == nil { // Игнорируем не-сообщения
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		// Обрабатываем команды
		switch update.Message.Command() {
		case "start":
			sendStartMessage(bot, update.Message.Chat.ID)
		case "help":
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Доступные команды:\n/start - Начать работу с ботом\n/catalog - Показать каталог автозапчастей\n/contacts - Контактная информация")
			bot.Send(msg)
		case "catalog":
			catalog, err := getCatalog(db)
			if err != nil {
				log.Printf("Failed to get catalog: %v", err)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка при получении каталога.")
				bot.Send(msg)
				continue
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, catalog)
			bot.Send(msg)
		case "contacts":
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Контактная информация:\nEmail: example@example.com\nTelegram: @example")
			bot.Send(msg)
		default:
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная команда. Используй /help для получения списка команд.")
			bot.Send(msg)
		}
	}
}

func createTables(db *sql.DB) {
	createPartsTable := `
    CREATE TABLE IF NOT EXISTS parts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        price REAL NOT NULL
    );
    `
	_, err := db.Exec(createPartsTable)
	if err != nil {
		log.Fatalf("Failed to create parts table: %v", err)
	}

	createOrdersTable := `
    CREATE TABLE IF NOT EXISTS orders (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        part_id INTEGER NOT NULL,
        quantity INTEGER NOT NULL,
        FOREIGN KEY (part_id) REFERENCES parts(id)
    );
    `
	_, err = db.Exec(createOrdersTable)
	if err != nil {
		log.Fatalf("Failed to create orders table: %v", err)
	}
}

func getCatalog(db *sql.DB) (string, error) {
	rows, err := db.Query("SELECT id, name, price FROM parts")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var catalog string
	for rows.Next() {
		var id int
		var name string
		var price float64
		err := rows.Scan(&id, &name, &price)
		if err != nil {
			return "", err
		}
		catalog += fmt.Sprintf("%d. %s - %.2f руб.\n", id, name, price)
	}

	return catalog, nil
}

func sendStartMessage(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Привет! Я бот для магазина автозапчастей. Выберите действие:")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("/catalog"),
			tgbotapi.NewKeyboardButton("/contacts"),
		),
	)
	bot.Send(msg)
}
