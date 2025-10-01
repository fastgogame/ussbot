package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	token := "8275968890:AAEJOmtFwzVSWuOPG5bMxl0qX9GktiBh-j4"
	if token == "" {
		log.Fatal("Укажи TOKEN в переменных окружения")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("✅ Authorized as %s", bot.Self.UserName)

	db, err := initDB("data/todo.db")
	if err != nil {
		log.Fatalf("initDB error: %v", err)
	}
	defer db.Close()

	log.Println("DB ready!")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		// ===== ГРУППА =====
		if update.Message != nil && (update.Message.Chat.IsGroup() || update.Message.Chat.IsSuperGroup()) {
			text := update.Message.Text

			if strings.Contains(text, "@") && strings.Contains(text, "+") {
				parts := strings.SplitN(text, "+", 2)
				if len(parts) == 2 {
					toUser := strings.TrimSpace(parts[0])
					taskText := strings.TrimSpace(parts[1])

					from := update.Message.From
					fromName := from.UserName
					if fromName == "" {
						fromName = from.FirstName
					}

					// Пытаемся найти assignee в БД
					var assigneeID int64
					err := db.QueryRow("SELECT user_id FROM members WHERE user_name = ?", toUser).Scan(&assigneeID)

					if err == sql.ErrNoRows {
						// Человека ещё нет в БД → просим его написать в личку
						msg := tgbotapi.NewMessage(update.Message.Chat.ID,
							fmt.Sprintf("📌 Задача для %s: %s\n(назначил @%s)\n\n👉 %s, напиши мне в личку /start, чтобы получать уведомления.",
								toUser, taskText, fromName, toUser))
						bot.Send(msg)
						continue
					} else if err != nil {
						log.Println("Ошибка поиска пользователя:", err)
						continue
					}

					// Создаём задачу
					task := Task{
						ChatID:       update.Message.Chat.ID,
						CreatorID:    int64(from.ID),
						CreatorName:  fromName,
						AssigneeID:   assigneeID,
						AssigneeName: toUser,
						Text:         taskText,
						CreatedAt:    time.Now(),
					}

					id, err := SaveTask(db, task)
					if err != nil {
						log.Printf("SaveTask error: %v", err)
						continue
					}

					log.Printf("Task saved with ID %d\n", id)

					// Ответ в группу
					//groupMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
					//	fmt.Sprintf("📌 Задача для %s: %s\n(назначил @%s)", toUser, taskText, fromName))
					//bot.Send(groupMsg)

					// Уведомления в личку
					bot.Send(tgbotapi.NewMessage(task.CreatorID, fmt.Sprintf("✅ Ты назначил задачу для %s: %s", toUser, taskText)))
					bot.Send(tgbotapi.NewMessage(task.AssigneeID, fmt.Sprintf("📥 Тебе назначена новая задача: %s (от @%s)", taskText, fromName)))
				}
			}
		}

		// ===== ЛИЧКА =====
		if update.Message != nil && update.Message.Chat.IsPrivate() {
			userID := update.Message.From.ID
			username := "@" + update.Message.From.UserName

			switch update.Message.Text {
			case "/start":
				// Сохраняем или обновляем участника в БД
				_, err := db.Exec(`
					INSERT INTO members (chat_id, user_id, user_name)
					VALUES (?, ?, ?)
					ON CONFLICT(chat_id, user_id) DO UPDATE SET user_name = excluded.user_name
				`, 0, userID, username)
				if err != nil {
					log.Println("Ошибка сохранения участника:", err)
				}

				// Обновляем старые задачи с пустым assignee_id
				res, err := db.Exec(`
					UPDATE tasks
					SET assignee_id = ?
					WHERE assignee_name = ? AND assignee_id = 0
				`, userID, username)
				if err != nil {
					log.Println("Ошибка обновления старых задач:", err)
				} else {
					n, _ := res.RowsAffected()
					if n > 0 {
						log.Printf("Привязано %d старых задач к %s", n, username)
					}
				}

				// Меню
				menu := tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("📋 Мои задачи"),
					),
				)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Привет! Теперь я могу присылать тебе задачи.\nВот меню:")
				msg.ReplyMarkup = menu
				bot.Send(msg)

			case "📋 Мои задачи":
				tasks, err := GetTasksByUser(db, int64(userID))
				if err != nil {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка при загрузке задач"))
					continue
				}
				if len(tasks) == 0 {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "У тебя пока нет задач 🎉"))
				} else {
					text := "Твои задачи:\n"
					for i, t := range tasks {
						text += fmt.Sprintf("%d. %s (от @%s)\n", i+1, t.Text, t.CreatorName)
					}
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, text))
				}

			default:
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Нажмите кнопку в меню 👇"))
			}
		}
	}
}
