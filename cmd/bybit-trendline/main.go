package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	"tinvest/internal/service/screener/bybitdensity/feed"
	"tinvest/internal/service/screener/bybitdensity/model"
	trendline "tinvest/internal/service/trading_strategy/bybit_trendline"

	tg "github.com/go-telegram/bot"
)

type checkpoint struct {
	TakeProfitR            float64
	RequirePreviousDayHigh bool
	MinRangePct            float64
	InPlayVersion          int
	Symbol                 string
	Minutes                int
	Fee, Slip              float64
	Start, Cursor          time.Time
}

func save(path string, s checkpoint) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err = os.WriteFile(path+".tmp", data, 0600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

var reportZone = time.FixedZone("МСК", 3*60*60)

func reportTime(t time.Time) string {
	return t.In(reportZone).Format("2006-01-02 15:04")
}

func report(path string, s checkpoint, end time.Time, e *engine, periods []inPlayPeriod) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Пробой наклонной · Bybit Spot %s · %d мин\n\nПериод МСК (UTC+3) [%s, %s). Только лонг. Комиссия %.3f%% и проскальзывание %.3f%% на каждую сторону.\n\n", s.Symbol, s.Minutes, reportTime(s.Start), reportTime(end), s.Fee*100, s.Slip*100)
	writeMetrics(&b, s.Start, end, e)
	if e.position != nil {
		fmt.Fprintf(&b, "На конец периода открыт лонг от %.8g, его результат исключён из статистики.\n\n", e.position.entry)
	}
	if e.pending != nil {
		b.WriteString("На последнем закрытии есть сигнал, ожидающий входа.\n\n")
	}
	fmt.Fprintf(&b, "Фильтр inplay проверяется каждые 30 минут. Средний размах 12 закрытых свечей M5 ≥ %g%%, средний объём выше среднего за будни предыдущей календарной недели.\n\n", s.MinRangePct)
	if s.RequirePreviousDayHigh {
		b.WriteString("Вход разрешён, если цена открытия свечи входа ≥ максимуму предыдущего календарного дня по Москве.\n\n")
	}
	writePeriods(&b, periods, end.Sub(s.Start))
	fmt.Fprintf(&b, "## Сделки\n\nВершины 3/15. Пробойная свеча открывается ниже линии и закрывается выше неё. Тело занимает не менее 65%% диапазона High–Low. Стоп на 0,01%% ниже Low свечи перед пробойной. Если расстояние от входа до стопа больше 0,2%%, вход пропускается. Тейк равен %gR от цены входа. Если внутри свечи задеты оба уровня, первым считается стоп. Время выхода обозначает конец свечи исполнения.\n\n| № | Сигнал МСК | Линия | Вход МСК | Цена | Стоп | Цель | Выход МСК | Цена выхода | Причина | Нетто |\n|---:|---|---|---|---:|---:|---:|---|---:|---|---:|\n", e.takeProfitR)
	for i, t := range e.done {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %.8g | %.8g | %.8g | %s | %.8g | %s | %+.2f%% |\n", i+1, reportTime(t.signal), t.scale, reportTime(t.opened), t.entry, t.stop, t.target, reportTime(t.closed), t.exit, t.reason, 100*t.net)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}
func run(ctx context.Context) (runErr error) {
	mode := flag.String("mode", "backtest", "backtest or watch")
	symbol := flag.String("symbol", "BTCUSDT", "Bybit spot symbol")
	minutes := flag.Int("minutes", 15, "candle interval")
	from := flag.String("from", "", "start date UTC")
	to := flag.String("to", "", "exclusive end date UTC")
	out := flag.String("out", "trendline-report.md", "Markdown output")
	state := flag.String("state", "", "watch checkpoint path")
	fee := flag.Float64("fee-bps", 10, "fee per side in basis points")
	slip := flag.Float64("slip-bps", 2, "slippage per side in basis points")
	flag.Parse()
	key := model.Key{Market: model.Spot, Symbol: strings.ToUpper(*symbol)}
	if !regexp.MustCompile("^[A-Z0-9]+$").MatchString(key.Symbol) {
		return errors.New("invalid symbol")
	}
	switch *minutes {
	case 1, 3, 5, 15, 30, 60, 120, 240, 360, 720:
	default:
		return errors.New("unsupported interval")
	}
	for _, v := range []float64{*fee, *slip} {
		if math.IsNaN(v) || v < 0 || v >= 10000 {
			return errors.New("invalid trading costs")
		}
	}
	d := time.Duration(*minutes) * time.Minute
	params := trendline.ParamsFor(key.Symbol)
	if params.TakeProfitR <= 0 || math.IsNaN(params.TakeProfitR) || math.IsInf(params.TakeProfitR, 0) {
		return errors.New("take profit multiple must be positive and finite")
	}
	threshold := params.InPlayMinRangePct
	s := checkpoint{TakeProfitR: params.TakeProfitR, RequirePreviousDayHigh: params.RequirePreviousDayHigh, MinRangePct: threshold, InPlayVersion: 1, Symbol: key.Symbol, Minutes: *minutes, Fee: *fee / 10000, Slip: *slip / 10000}
	var end time.Time
	var send func(string) error
	path := *state
	switch *mode {
	case "watch":
		token := os.Getenv("TELEGRAM_BOT_TOKEN")
		chat, err := strconv.ParseInt(os.Getenv("TELEGRAM_GROUP_CHAT_ID"), 10, 64)
		topic, topicErr := strconv.Atoi(os.Getenv("TELEGRAM_TOPIC_TRENDLINE"))
		if token == "" || err != nil || chat == 0 || topicErr != nil || topic <= 0 {
			return errors.New("set TELEGRAM_BOT_TOKEN, TELEGRAM_GROUP_CHAT_ID and TELEGRAM_TOPIC_TRENDLINE")
		}
		bot, err := tg.New(token, tg.WithSkipGetMe())
		if err != nil {
			return errors.New("invalid Telegram configuration")
		}
		send = func(msg string) error {
			req, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			_, err := bot.SendMessage(req, &tg.SendMessageParams{ChatID: chat, MessageThreadID: topic, Text: msg})
			if err != nil {
				return errors.New("failed to deliver Telegram message; checkpoint retained")
			}
			return nil
		}
		if path == "" {
			path = fmt.Sprintf("trendline-%s-%dm.json", key.Symbol, *minutes)
		}
		lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return err
		}
		defer func() {
			if err := lock.Close(); err != nil {
				runErr = errors.Join(runErr, err)
			}
		}()
		if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			return errors.New("watch checkpoint already in use")
		}
		raw, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			s.Start = time.Now().UTC().Truncate(d)
			s.Cursor = s.Start
			if err = save(path, s); err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			var old checkpoint
			if err = json.Unmarshal(raw, &old); err != nil {
				return err
			}
			if old.InPlayVersion != s.InPlayVersion {
				return errors.New("checkpoint predates inplay filtering; use a new -state path")
			}
			if old.MinRangePct == 0 {
				old.MinRangePct = 1 // Older checkpoints always used 1%.
			}
			if old.MinRangePct != threshold {
				return fmt.Errorf("checkpoint uses inplay threshold %g%%, configured %g%%; use a new -state path", old.MinRangePct, threshold)
			}
			if old.TakeProfitR != s.TakeProfitR {
				return errors.New("take profit setting changed; use a new -state path")
			}
			if old.RequirePreviousDayHigh != s.RequirePreviousDayHigh {
				return errors.New("previous day high filter changed; use a new -state path")
			}
			if old.Symbol != s.Symbol || old.Minutes != s.Minutes || old.Fee != s.Fee || old.Slip != s.Slip || old.Start.IsZero() || old.Cursor.Before(old.Start) {
				return errors.New("incompatible checkpoint")
			}
			s = old
		}
	case "backtest":
		var err error
		s.Start, err = time.Parse(time.DateOnly, *from)
		if err != nil {
			return errors.New("supply -from YYYY-MM-DD")
		}
		end, err = time.Parse(time.DateOnly, *to)
		if err != nil || !s.Start.Before(end) || end.After(time.Now().UTC().Truncate(d)) {
			return errors.New("invalid exclusive -to date")
		}
	default:
		return errors.New("unknown mode")
	}
	client := feed.NewClient("", "")
	filter := inPlayFilter{minRangePct: threshold, load: client.SpotM5, key: key}
	e := newEngine(s.Fee, s.Slip, params.TakeProfitR)
	var highDay time.Time
	var previousHigh float64
	var periods []inPlayPeriod
	cursor := s.Start.Add(-650 * d)
replay:
	for ctx.Err() == nil {
		limit := end
		if send != nil {
			limit = time.Now().UTC().Truncate(d)
		}
		next := cursor.Add(min(1000*d, 7*24*time.Hour))
		if next.After(limit) {
			next = limit
		}
		from := cursor
		if from.Before(s.Start) {
			from = s.Start
		}
		checkTo := next
		if send != nil && next.Equal(limit) {
			checkTo = time.Now().UTC().Truncate(checkInterval)
			if checkTo.Before(next) {
				checkTo = next
			}
		}
		if !checkTo.Before(from) {
			if err := filter.update(ctx, from, checkTo); err != nil {
				if send == nil {
					return err
				}
				fmt.Fprintln(os.Stderr, err)
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(5 * time.Second):
					continue
				}
			}
		}
		if send == nil && from.Before(next) {
			periods = appendPeriods(periods, &filter, from, next)
		}
		if cursor.Before(limit) {
			candles, err := client.SpotBars(ctx, key, d, cursor, next)
			if err != nil {
				if send == nil {
					return err
				}
				fmt.Fprintln(os.Stderr, err)
			} else {
				for _, c := range candles {
					opened := c.End.Add(-d)
					allowed := filter.allows(opened)
					if s.RequirePreviousDayHigh && allowed && e.pending != nil {
						local := opened.In(reportZone)
						day := time.Date(local.Year(), local.Month(), local.Day()-1, 0, 0, 0, 0, reportZone)
						if !day.Equal(highDay) {
							high, err := client.DayHigh(ctx, key, day)
							if err != nil {
								if send == nil {
									return err
								}
								fmt.Fprintln(os.Stderr, err)
								cursor = opened
								select {
								case <-ctx.Done():
									return nil
								case <-time.After(5 * time.Second):
								}
								continue replay
							}
							highDay, previousHigh = day, high
						}
						e.entryFloor = previousHigh
					}
					events := e.step(c, d, allowed, c.End.After(s.Start) && filter.allows(c.End))
					if send != nil && c.End.After(s.Cursor) {
						if len(events) > 0 {
							msg := fmt.Sprintf("СИМУЛЯЦИЯ · Bybit Spot %s · %d мин · %s UTC\n%s", key.Symbol, *minutes, c.End.Format("2006-01-02 15:04"), strings.Join(events, "\n"))
							if err := send(msg); err != nil {
								return err
							}
						}
						s.Cursor = c.End
						if err := save(path, s); err != nil {
							return err
						}
					}
					if send != nil {
						e.done = nil
					}
				}
				cursor = next
				continue
			}
		} else if send == nil {
			return report(*out, s, end, e, periods)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(5 * time.Second):
		}
	}
	return ctx.Err()
}
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
