package telegram_commands

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"

	"tinvest/internal/service/portfolio/analyze"
	"tinvest/internal/service/portfolio/yield"
	"tinvest/internal/service/trading_strategy/bonds"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

// SenderFactory строит Client, привязанный к чату/теме, откуда пришла команда.
type SenderFactory func(chatID int64, threadID int) telegram.Client

type Commands struct {
	analyze        analyze.Analyze
	yield          yield.Yield
	bonds          bonds.Bonds
	newSender      SenderFactory
	allowedUserIDs map[int64]struct{}

	mu      sync.Mutex
	running map[string]bool
}

func New(a analyze.Analyze, y yield.Yield, b bonds.Bonds, f SenderFactory, allowed []int64) *Commands {
	ids := make(map[int64]struct{}, len(allowed))
	for _, id := range allowed {
		ids[id] = struct{}{}
	}

	return &Commands{
		analyze:        a,
		yield:          y,
		bonds:          b,
		newSender:      f,
		allowedUserIDs: ids,
		running:        make(map[string]bool),
	}
}

const helpText = `Доступные команды:
/bonds_portfolio — распределение облигаций в портфеле
/yield — доходность портфеля YTD (XIRR)
/bonds_screener — скринер облигаций к покупке
/help — этот список`

// Handle обрабатывает одно входящее сообщение. Возвращает false, если оно
// проигнорировано (не-whitelisted пользователь или неизвестная команда).
// Команды от чужих пользователей игнорируются молча.
func (c *Commands) Handle(ctx context.Context, text string, chatID int64, threadID int, userID int64) bool {
	if _, ok := c.allowedUserIDs[userID]; !ok {
		return false
	}
	cmd, _, _ := strings.Cut(strings.TrimSpace(text), " ")
	cmd, _, _ = strings.Cut(cmd, "@") // "/yield@MyBot" -> "/yield"
	tg := c.newSender(chatID, threadID)

	switch cmd {
	case "/help", "/start":
		_ = tg.SendMessage(helpText)
	case "/bonds_portfolio":
		c.runExclusive(ctx, cmd, tg, func(ctx context.Context) error {
			return c.analyze.BondsPortfolio(ctx, tg)
		})
	case "/yield":
		c.runExclusive(ctx, cmd, tg, func(ctx context.Context) error {
			return c.yield.PortfolioYieldYTD(ctx, tg)
		})
	case "/bonds_screener":
		c.runExclusive(ctx, cmd, tg, func(ctx context.Context) error {
			return c.bonds.Trade(ctx, tg)
		})
	default:
		return false
	}

	return true
}

// runExclusive подтверждает приём, выполняет расчёт в горутине и не
// допускает второй параллельный запуск той же команды.
func (c *Commands) runExclusive(ctx context.Context, cmd string, tg telegram.Client, fn func(context.Context) error) {
	c.mu.Lock()
	if c.running[cmd] {
		c.mu.Unlock()
		_ = tg.SendMessage("⏳ " + cmd + " уже выполняется")

		return
	}
	c.running[cmd] = true
	c.mu.Unlock()

	_ = tg.SendMessage("⏳ Считаю " + cmd + "…")

	go func() {
		// Один defer на снятие флага и recover: флаг снимается и при панике,
		// причём до отправки «❌» — к моменту ответа команда снова доступна.
		// Recover обязателен: паника в analyze/yield/bonds иначе уронит весь
		// процесс вместе с live-торговыми горутинами (прецедент guard'а —
		// golden_x.Trade).
		defer func() {
			c.mu.Lock()
			c.running[cmd] = false
			c.mu.Unlock()
			if r := recover(); r != nil {
				logger.ErrorContext(ctx, "telegram command panicked", fmt.Sprintf("%s: %v\n%s", cmd, r, debug.Stack()))
				_ = tg.SendMessage("❌ " + cmd + ": внутренняя ошибка, подробности в логах")
			}
		}()
		if err := fn(ctx); err != nil {
			logger.ErrorContext(ctx, "telegram command failed", err.Error())
			_ = tg.SendMessage("❌ " + cmd + ": ошибка выполнения, подробности в логах")
		}
	}()
}
