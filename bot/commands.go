package bot

import (
	"context"

	tgbotapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// processCommand routes command to the method that handles it
func (b *Bot) processCommand(ctx context.Context, update *models.Update) {
	b.logger.Infow("Command Received", "from", update.Message.From.Username, "command", update.Message.Text)

	switch update.Message.Text {
	case "/start":
		b.processStartCommand(ctx, update)
	case "/help":
		b.processHelpCommand(ctx, update)
	default:
		b.processUnknownCommand(ctx, update)
	}
}

// processStartCommand creates user in the database if user does not exist and sends starting message to the user
func (b *Bot) processStartCommand(ctx context.Context, update *models.Update) {
	//Send starting message
	if _, err := b.b.SendMessage(ctx, &tgbotapi.SendMessageParams{ChatID: update.Message.Chat.ID, Text: b.msgs.startMsg[language(update.Message.From)]}); err != nil {
		b.logger.Errorw("error sending message", "error", err)
		return
	}
}

// processHelpCommand sends user the list of the available commands
func (b *Bot) processHelpCommand(ctx context.Context, update *models.Update) {
	if _, err := b.b.SendMessage(ctx, &tgbotapi.SendMessageParams{ChatID: update.Message.Chat.ID, Text: b.msgs.helpMsg[language(update.Message.From)]}); err != nil {
		b.logger.Errorw("error sending message", "error", err)
	}
}

// processUnknownCommand sends user the message stating that the bot does not know this command
func (b *Bot) processUnknownCommand(ctx context.Context, update *models.Update) {
	if _, err := b.b.SendMessage(ctx, &tgbotapi.SendMessageParams{ChatID: update.Message.Chat.ID, Text: b.msgs.unknownMsg[language(update.Message.From)]}); err != nil {
		b.logger.Errorw("error sending message", "error", err)
	}
}
