package bot

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/go-telegram/bot"
	tgbotapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	english = "en"
	russian = "ru"
)

type Bot struct {
	b      *tgbotapi.Bot
	logger *zap.SugaredLogger
	msgs   *messages
}

// New creates a new bot
func New(token string, logger *zap.SugaredLogger) (*Bot, error) {
	//Create bot using provided dependencies
	bot := &Bot{logger: logger}

	//Create telegram bot with a default handler
	b, err := tgbotapi.New(token, tgbotapi.WithDefaultHandler(bot.defaultHandler))
	if err != nil {
		return nil, err
	}
	bot.b = b
	bot.msgs = loadMessages()
	return bot, nil
}

// Run runs the bot using long polling
func (b *Bot) Run(ctx context.Context) {
	b.b.Start(ctx)
}

// RunWebhook runs bot using webhook
func (b *Bot) RunWebhook(ctx context.Context, address string) error {
	//delete webhook before shutdown
	defer func() {
		if _, err := b.b.DeleteWebhook(context.Background(), &tgbotapi.DeleteWebhookParams{DropPendingUpdates: true}); err != nil {
			panic(err)
		}
	}()
	go b.b.StartWebhook(ctx)

	//Set tup server for the webhook
	//Create http server for the webhook
	srv := &http.Server{
		Addr:    address,
		Handler: b.b.WebhookHandler(),
	}

	//Create channel for errors
	ch := make(chan error)

	//Run server in a goroutine
	go func() {
		defer close(ch)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			ch <- err
			return
		}
		ch <- nil
	}()

	//Wait for either shutdown or an error
	select {
	case <-ctx.Done():
		if err := srv.Shutdown(context.Background()); err != nil {
			return err
		}
		err := <-ch
		if err != nil {
			return err
		}
	case err := <-ch:
		return err
	}
	return nil
}

// defaultHandler routes request to the bot
func (b *Bot) defaultHandler(ctx context.Context, _ *bot.Bot, update *models.Update) {
	//Check if the update is a preCheckoutQuery, callbackQuery or message
	switch {
	case update.Message != nil && strings.HasPrefix(update.Message.Text, "/"):
		b.processCommand(ctx, update)
	case update.Message != nil && update.Message.Video != nil:
		b.processVideo(ctx, update)
	default:
		b.processUnknownMessage(ctx, update)
	}
}

// processUnknownCommand sends user the message stating that the bot does not know this command
func (b *Bot) processUnknownMessage(ctx context.Context, update *models.Update) {
	if _, err := b.b.SendMessage(ctx, &tgbotapi.SendMessageParams{ChatID: update.Message.Chat.ID, Text: b.msgs.unknownMsg[language(update.Message.From)]}); err != nil {
		b.logger.Errorw("error sending message", "error", err)
	}
}

// Returns user's language code if its russian or english, else returns english
func language(user *models.User) string {
	switch user.LanguageCode {
	case "ru":
		return russian
	default:
		return english
	}
}

type messages struct {
	startMsg         map[string]string
	helpMsg          map[string]string
	errorMsg         map[string]string
	unknownMsg       map[string]string
	videoTooLargeMsg map[string]string
	waitMsg          map[string]string
}

func loadMessages() *messages {
	return &messages{
		startMsg: map[string]string{
			"en": "Hi! Send me a video, and I’ll convert it into a video message (circle format) for you.",
			"ru": "Привет! Отправь мне видео, и я сделаю из него видеосообщение (в кружочке) для тебя.",
		},
		errorMsg: map[string]string{
			"en": "⚠️ Something went wrong. Wrong file format or internal server error.",
			"ru": "⚠️ Что-то пошло не так. Неверный формат файла или ошибка сервера.",
		},
		unknownMsg: map[string]string{
			"en": "I can only process videos. Please send me a video file.",
			"ru": "Я могу обрабатывать только видео. Пожалуйста, отправьте мне видеофайл.",
		},
		helpMsg: map[string]string{
			"en": "Send me a video, and I’ll convert it into a video message (circle format) for you.\nThis is my only function.\nIf you have any questions contact the creator of the bot: @dafraer.",
			"ru": "Отправь мне видео, и я сделаю из него видеосообщение (в кружочке) для тебя.\n Это моя единственная функция.\nЕсли у вас имеются дополнительные вопросы, пишите автору бота: @dafraer.",
		},
		videoTooLargeMsg: map[string]string{
			"en": "The video you sent is too large. Please send a smaller file.",
			"ru": "Отправленное вами видео слишком большое. Пожалуйста, отправьте файл поменьше.",
		},
		waitMsg: map[string]string{
			"en": "Your video note is being generated, please wait…",
			"ru": "Ваш кружок генерируется, пожалуйста, подождите…",
		},
	}
}

// sendErrorMessage sends error message to the user
func (b *Bot) sendErrorMessage(ctx context.Context, update *models.Update) {
	if _, err := b.b.SendMessage(ctx, &tgbotapi.SendMessageParams{ChatID: update.Message.Chat.ID, Text: b.msgs.errorMsg[language(update.Message.From)]}); err != nil {
		b.logger.Errorw("error sending message", "error", err)
	}
}
