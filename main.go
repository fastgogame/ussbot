package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const botToken = "8275968890:AAEJOmtFwzVSWuOPG5bMxl0qX9GktiBh-j4"

func main() {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	bot.Debug = true
	log.Printf("Бот авторизован как: %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	log.Println("Бот запущен! Добавьте его в группу.")

	for {
		select {
		case update := <-updates:
			handleUpdate(bot, update)
		case <-stop:
			log.Println("Останавливаем бота...")
			return
		}
	}
}

// Создаем ReplyKeyboard меню
func getMainMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🕐 Текущее время"),
			tgbotapi.NewKeyboardButton("🌍 Часовой пояс"),
			tgbotapi.NewKeyboardButton("👋 Поприветствовать"),
		),
	)
}

func handleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	// Обрабатываем добавление бота в группу
	if update.Message != nil && update.Message.NewChatMembers != nil {
		handleBotAddedToGroup(bot, update.Message)
		return
	}

	// Обрабатываем текстовые сообщения (кнопки меню)
	if update.Message != nil && update.Message.Text != "" {
		handleTextMessage(bot, update.Message)
		return
	}
}

func handleBotAddedToGroup(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	for _, newMember := range message.NewChatMembers {
		if newMember.ID == bot.Self.ID {
			welcomeText := `👋 Привет всем! Я новый бот этой группы! 

Используйте меню внизу для взаимодействия со мной.`

			msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
			msg.ReplyMarkup = getMainMenu()
			sendMessage(bot, msg)
			log.Printf("Бот представился в группе: %s", message.Chat.Title)
			return
		}
	}
}

func handleTextMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// Игнорируем сообщения от самого бота
	if message.From != nil && message.From.ID == bot.Self.ID {
		return
	}

	switch message.Text {
	case "🕐 Текущее время":
		handleTimeCommand(bot, message)
	case "🌍 Часовой пояс":
		handleTimezoneCommand(bot, message)
	case "👋 Поприветствовать":
		handleSayHello(bot, message)
	default:
		return
	}
}

func handleSayHello(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	instructions := `👋 **Приветствие пользователя**

Напишите @username пользователя, которого хотите поприветствовать.

**Пример:**
@username
или
@ivanov`

	msg := tgbotapi.NewMessage(message.Chat.ID, instructions)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true) // Убираем меню для чистого ввода
	sendMessage(bot, msg)
}

func handleTimeCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// Текущее время в Москве
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		location = time.UTC
	}

	currentTime := time.Now().In(location)

	// Форматируем время
	timeStr := currentTime.Format("02.01.2006 15:04:05")
	timezoneStr := currentTime.Format("MST")

	text := "🕐 **Текущее время:**\n\n" +
		"**Дата и время:** " + timeStr + "\n" +
		"**Часовой пояс:** " + timezoneStr + " (Europe/Moscow)"

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = getMainMenu()
	sendMessage(bot, msg)
}

func handleTimezoneCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// Текущий часовой пояс
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		location = time.UTC
	}

	currentTime := time.Now().In(location)
	timezoneStr := currentTime.Format("MST")
	timezoneOffset := currentTime.Format("-07:00")

	text := "🌍 **Текущий часовой пояс:**\n\n" +
		"**Название:** Europe/Moscow\n" +
		"**Смещение:** " + timezoneStr + " (" + timezoneOffset + ")\n" +
		"**Город:** Москва, Россия\n\n" +
		"_Время отображается по московскому времени_"

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = getMainMenu()
	sendMessage(bot, msg)
}

// Удаляем сообщение
func deleteMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	_, err := bot.Send(deleteMsg)
	if err != nil {
		log.Printf("Ошибка удаления сообщения: %v", err)
	}
}

func sendMessage(bot *tgbotapi.BotAPI, msg tgbotapi.MessageConfig) {
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки: %v", err)
	}
}
