package telegram_commands

import (
	"context"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"tinvest/pkg/client/telegram"
)

// Listener подписывает роутер на входящие сообщения и держит long-polling.
type Listener struct {
	bot      *telegram.Bot
	commands *Commands
}

func NewListener(b *telegram.Bot, c *Commands) *Listener {
	return &Listener{bot: b, commands: c}
}

// Run блокируется до отмены ctx. Переподключение при сетевых ошибках
// long-polling библиотека go-telegram/bot выполняет сама.
func (l *Listener) Run(ctx context.Context) {
	l.bot.API().RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && strings.HasPrefix(update.Message.Text, "/")
	}, l.handle)
	l.bot.API().Start(ctx)
}

func (l *Listener) handle(ctx context.Context, _ *tgbot.Bot, update *models.Update) {
	m := update.Message
	if m == nil || m.From == nil {
		return
	}
	l.commands.Handle(ctx, m.Text, m.Chat.ID, m.MessageThreadID, m.From.ID)
}
