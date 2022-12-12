package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var baseMarkup = tgbotapi.NewInlineKeyboardRow(
	tgbotapi.NewInlineKeyboardButtonData("Изменить пожелания", "/description"),
	tgbotapi.NewInlineKeyboardButtonData("Моя карточка", "/me"),
)

var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	baseMarkup,
)

var adminKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	baseMarkup,
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Начать тайного дед мороза!", "/start_game"),
	),
)

var users = make(map[int64]User)
var bot *tgbotapi.BotAPI
var errorChan = make(chan BotRuntimeError)

type BotRuntimeError struct {
	err    error
	chatId int64
}
type User struct {
	ChatId      int64  `json:"chat_id"`
	UserName    string `json:"user_name"`
	FirstName   string `json:"first_name"`
	InviteLink  string `json:"invite_link"`
	State       int    `json:"state"`
	Description string `json:"description"`
	Recipient   int64  `json:"recipient"`
}

func (u *User) getRecipientInfo() string {
	if recipient, ok := users[u.Recipient]; ok {

		return "-Пожелания: " + recipient.Description +
			"\n-Чат айди: " + strconv.FormatInt(recipient.ChatId, 10) +
			"\n-Ссылка: " + recipient.InviteLink +
			"\n-Username: @" + recipient.UserName +
			"\n-Имя: " + recipient.FirstName
	}

	return "Отсутствует"
}
func (u *User) toString() string {
	return "1)Мои пожелания: " + u.Description +
		"\n2)Айди чата: " + strconv.FormatInt(u.ChatId, 10) +
		"\n2)Username: @" + u.UserName +
		"\n4)Имя: " + u.FirstName +
		"\n5)Информация о цели:\n" + u.getRecipientInfo()
}

var start = func(update tgbotapi.Update) (msg tgbotapi.MessageConfig, err error) {
	var message *tgbotapi.Message

	if update.Message != nil {
		message = update.Message
	} else {
		message = update.CallbackQuery.Message
	}

	if _, ok := users[message.Chat.ID]; ok {
		msg = tgbotapi.NewMessage(message.Chat.ID, "Вы уже участвуете в игре!")
		msg.ReplyMarkup = numericKeyboard
		return msg, nil
	}
	var inviteLink = ""

	if message.Chat.UserName != "" {
		inviteLink = "https://t.me/" + message.Chat.UserName
	}

	var user = User{
		message.Chat.ID,
		message.Chat.UserName,
		message.Chat.FirstName,
		inviteLink,
		0,
		"Пока здесь пусто :(",
		0,
	}

	go saveUser(user)

	msg = tgbotapi.NewMessage(message.Chat.ID, "Отлично, вы записаны!")
	msg.ReplyMarkup = numericKeyboard

	return msg, nil
}
var me = func(update tgbotapi.Update) (msg tgbotapi.MessageConfig, err error) {
	var chatId int64

	if update.Message != nil {
		chatId = update.Message.Chat.ID
	} else {
		chatId = update.CallbackQuery.Message.Chat.ID
	}

	if user, ok := users[chatId]; ok {
		if user.Description == "" {
			user.Description = "Здесь пока пусто :("
		}

		msg = tgbotapi.NewMessage(chatId, user.toString())

		return msg, nil
	}

	return tgbotapi.NewMessage(
		chatId,
		"Упс! Не нашли вас в участниках.\nСначала запишитесь! /start",
	), nil
}
var changeDescription = func(update tgbotapi.Update) (msg tgbotapi.MessageConfig, err error) {
	var chatId int64

	if update.Message != nil {
		chatId = update.Message.Chat.ID
	} else {
		chatId = update.CallbackQuery.Message.Chat.ID
	}

	if user, ok := users[chatId]; ok {
		user.State = 1

		saveUser(user)

		msg = tgbotapi.NewMessage(chatId, "Что бы вы хотели на новый год?)")
		return msg, nil
	}

	return tgbotapi.NewMessage(
		chatId,
		"Упс! Не нашли вас в участниках.\nСначала запишитесь! /start",
	), nil
}
var showKeyboard = func(update tgbotapi.Update) (msg tgbotapi.MessageConfig, err error) {
	var chatId int64

	if update.Message != nil {
		chatId = update.Message.Chat.ID
	} else {
		chatId = update.CallbackQuery.Message.Chat.ID
	}

	msg = tgbotapi.NewMessage(
		chatId,
		"Чи шо делаем?)",
	)

	if val, err := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64); val == chatId && err == nil {
		msg.ReplyMarkup = adminKeyboard
	} else if err == nil {
		msg.ReplyMarkup = numericKeyboard
	} else if err != nil {
		errorChan <- BotRuntimeError{
			chatId: chatId,
			err:    errors.New("Ошибка на сервере. Error №02\n"),
		}
		fmt.Println(err)
	}

	return msg, nil
}
var startGame = func(update tgbotapi.Update) (msg tgbotapi.MessageConfig, err error) {
	var chatId int64

	if update.Message != nil {
		chatId = update.Message.Chat.ID
	} else {
		chatId = update.CallbackQuery.Message.Chat.ID
	}

	if val, err := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64); val == chatId && err == nil {
		count := len(users)

		var beforeId int64 = 0

		var firstId int64 = 0
		var i = 0
		for key, element := range users {
			i++

			if i == count {
				if user, ok := users[firstId]; ok {
					user.Recipient = element.ChatId
					users[firstId] = user
				}
			}

			if beforeId == 0 {
				firstId = element.ChatId
				beforeId = element.ChatId
				continue
			}

			element.Recipient = beforeId
			beforeId = element.ChatId
			users[key] = element
		}

		go saveUsers()

		msg = tgbotapi.NewMessage(chatId, "Успешно!")
		return msg, nil
	} else if err != nil {
		errorChan <- BotRuntimeError{
			chatId: chatId,
			err:    errors.New("Ошибка на сервере. Error №02\n"),
		}
		fmt.Println(err)
	}

	msg = tgbotapi.NewMessage(chatId, "У вас нет прав!")
	return msg, nil
}
var terminator = map[string]func(update tgbotapi.Update) (msg tgbotapi.MessageConfig, err error){
	"/start":       start,
	"/me":          me,
	"/description": changeDescription,
	"/keyboard":    showKeyboard,
	"/start_game":  startGame,
}

func gopherErrorHandler() {
	for {
		select {
		case runtimeError := <-errorChan:
			{
				msg := tgbotapi.NewMessage(runtimeError.chatId, runtimeError.err.Error())
				_, err := bot.Send(msg)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
}
func saveUser(user User) {
	users[user.ChatId] = user

	data, _ := json.Marshal(users)

	if os.WriteFile("users.json", data, 0777) != nil {
		errorChan <- BotRuntimeError{
			chatId: user.ChatId,
			err:    errors.New("Ошибка сервера! Повторите попытку записи позже.\n Error №01"),
		}
		log.Println("Write error!")
	}
}
func saveUsers() {
	data, _ := json.Marshal(users)
	adminId, err := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	if os.WriteFile("out.json", data, 0777) != nil {
		errorChan <- BotRuntimeError{
			chatId: adminId,
			err:    err,
		}
		log.Println("Write error!")
		return
	}
	errorChan <- BotRuntimeError{
		chatId: adminId,
		err:    errors.New("Удачно записано в файл\n"),
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	body, err := os.ReadFile("users.json")
	if err != nil {
		log.Fatalf("unable to read file: %v", err)
	}

	json.Unmarshal(body, &users)

	fmt.Println(users)

	go gopherErrorHandler()

	bot, err = tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_KEY"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			if !update.Message.IsCommand() {
				if user, ok := users[update.Message.Chat.ID]; ok {
					msg := tgbotapi.MessageConfig{}

					switch user.State {
					case 1:
						{
							user.Description = update.Message.Text
							user.State = 0
							go saveUser(user)
							msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Отлично! Ваше описание добавлено")
							msg.ReplyMarkup = numericKeyboard
						}

					default:
						{
							msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Я не понял чи шо делать(")
						}
					}

					if _, err = bot.Send(msg); err != nil {
						log.Fatal(err)
					}
				} else {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID,
						"Вы пока не участвуете! "+
							"Для старата введите: /start")

					if _, err = bot.Send(msg); err != nil {
						log.Fatal(err)
					}
				}
				continue
			}

			if function, ok := terminator[update.Message.Text]; ok {
				msg, err := function(update)

				if err != nil {
					errorChan <- BotRuntimeError{chatId: update.Message.Chat.ID, err: err}
					continue
				}

				if _, err = bot.Send(msg); err != nil {
					log.Fatal(err)
				}
			} else {
				errorChan <- BotRuntimeError{
					chatId: update.Message.Chat.ID,
					err:    errors.New("Неизвестная комманда!\n"),
				}
			}
		} else if update.CallbackQuery != nil {
			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)

			if _, err := bot.Request(callback); err != nil {
				log.Fatal(err)
			}

			if function, ok := terminator[update.CallbackQuery.Data]; ok {
				msg, err := function(update)

				if err != nil {
					errorChan <- BotRuntimeError{chatId: update.CallbackQuery.Message.Chat.ID, err: err}
					continue
				}

				if _, err = bot.Send(msg); err != nil {
					log.Fatal(err)
				}
			} else {
				errorChan <- BotRuntimeError{
					chatId: update.CallbackQuery.Message.Chat.ID,
					err:    errors.New("Неизвестная комманда!\n"),
				}
			}
		}
	}
}
