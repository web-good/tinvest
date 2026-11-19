package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"tinvest/internal/service/screener/bybitdensity/strategy"

	"github.com/joho/godotenv"
)

const envPrefix = "BYBIT_DENSITY_"

type options struct {
	strategy      strategy.Config
	stateDir      string
	spotSymbols   []string
	linearSymbols []string
	symbolLimit   int
	depth         int
	screener      telegramDestination
	simulation    telegramDestination
	telegramToken string
	checkFeed     bool
	checkTimeout  time.Duration
	restURL       string
	wsURL         string
}

func readOptions(args []string, out io.Writer) (options, error) {
	op := options{strategy: strategy.DefaultConfig()}
	cfg := &op.strategy
	fs := flag.NewFlagSet("bybit-density", flag.ContinueOnError)
	fs.SetOutput(out)
	envFile := fs.String("env-file", "", "optional dotenv file")
	spot := fs.String("spot-symbols", "", "comma-separated spot USDT pairs; empty selects by turnover")
	linear := fs.String("linear-symbols", "", "comma-separated linear USDT perpetuals; empty selects by turnover")
	fs.StringVar(&op.stateDir, "state-dir", "reports/bybit-density", "local simulation state directory")
	fs.IntVar(&op.symbolLimit, "symbol-limit", 20, "number of automatic selections per market")
	fs.IntVar(&op.depth, "depth", 200, "Bybit order book depth per side")
	fs.Int64Var(&op.screener.chatID, "screener-chat-id", 0, "Telegram chat for confirmed densities")
	fs.IntVar(&op.screener.topicID, "screener-topic-id", 0, "optional Telegram topic for confirmed densities")
	fs.Int64Var(&op.simulation.chatID, "simulation-chat-id", 0, "Telegram chat for simulated fills")
	fs.IntVar(&op.simulation.topicID, "simulation-topic-id", 0, "optional Telegram topic for simulated fills")
	fs.BoolVar(&op.checkFeed, "check-feed", false, "verify public BTCUSDT feeds in both markets and exit")
	fs.DurationVar(&op.checkTimeout, "check-timeout", 90*time.Second, "timeout for public feed check")
	fs.StringVar(&op.restURL, "rest-url", "", "override the Bybit REST base URL")
	fs.StringVar(&op.wsURL, "ws-url", "", "override the Bybit WebSocket base URL")
	fs.Float64Var(&cfg.MinNotional, "min-notional", cfg.MinNotional, "minimum density notional in USDT")
	fs.Float64Var(&cfg.MinMultiple, "min-multiple", cfg.MinMultiple, "minimum multiple of neighbor median")
	fs.Float64Var(&cfg.MaxDistance, "max-distance", cfg.MaxDistance, "maximum price distance as a fraction")
	fs.DurationVar(&cfg.Persistence, "persistence", cfg.Persistence, "minimum density confirmation time")
	fs.DurationVar(&cfg.Freshness, "freshness", cfg.Freshness, "maximum age of a book update")
	fs.Float64Var(&cfg.PaperNotional, "paper-notional", cfg.PaperNotional, "maximum virtual entry notional in USDT")
	fs.Float64Var(&cfg.SpotFee, "spot-fee", cfg.SpotFee, "simulated spot commission per side as a fraction")
	fs.Float64Var(&cfg.LinearFee, "linear-fee", cfg.LinearFee, "simulated perpetual commission per side as a fraction")
	fs.Int64Var(&cfg.StopTicks, "stop-ticks", cfg.StopTicks, "stop distance behind density in price ticks")
	fs.Int64Var(&cfg.EntryZoneTicks, "entry-zone-ticks", cfg.EntryZoneTicks, "maximum pinbar distance toward the market in price ticks")
	fs.Float64Var(&cfg.RewardRisk, "reward-risk", cfg.RewardRisk, "take profit as a multiple of stop distance")
	fs.Float64Var(&cfg.MinTarget, "min-target", cfg.MinTarget, "minimum take profit distance as a fraction")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if fs.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected positional arguments")
	}
	var dotenv map[string]string
	if *envFile != "" {
		var err error
		dotenv, err = godotenv.Read(*envFile)
		if err != nil {
			return options{}, fmt.Errorf("read dotenv: %w", err)
		}
	}
	lookup := func(name string) (string, bool) {
		if value, ok := os.LookupEnv(name); ok {
			return value, true
		}
		value, ok := dotenv[name]
		return value, ok
	}
	set := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	var envErr error
	fs.VisitAll(func(f *flag.Flag) {
		if set[f.Name] || f.Name == "env-file" || envErr != nil {
			return
		}
		name := envPrefix + strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
		if value, ok := lookup(name); ok {
			if err := f.Value.Set(value); err != nil {
				envErr = fmt.Errorf("%s: %w", name, err)
			}
		}
	})
	if envErr != nil {
		return options{}, envErr
	}
	op.telegramToken, _ = lookup(envPrefix + "TELEGRAM_TOKEN")
	op.telegramToken = strings.TrimSpace(op.telegramToken)
	if op.telegramToken == "" {
		token, _ := lookup("TELEGRAM")
		op.telegramToken = strings.TrimSpace(token)
	}
	op.spotSymbols = splitSymbols(*spot)
	op.linearSymbols = splitSymbols(*linear)
	return op, op.validate()
}

func splitSymbols(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	items := strings.Split(input, ",")
	for index := range items {
		items[index] = strings.ToUpper(strings.TrimSpace(items[index]))
	}
	return items
}

func (op options) validate() error {
	switch {
	case op.symbolLimit < 1 || op.symbolLimit > 100:
		return fmt.Errorf("symbol-limit must be between 1 and 100 per market")
	case len(op.spotSymbols) > 100 || len(op.linearSymbols) > 100:
		return fmt.Errorf("explicit instrument lists must have at most 100 entries per market")
	case op.depth != 50 && op.depth != 200 && op.depth != 1000:
		return fmt.Errorf("depth must be 50, 200 or 1000")
	case op.checkTimeout <= 0:
		return fmt.Errorf("check-timeout must be positive")
	case !op.checkFeed && op.telegramToken == "":
		return fmt.Errorf("set TELEGRAM or %sTELEGRAM_TOKEN", envPrefix)
	case !op.checkFeed && (op.screener.chatID == 0 || op.simulation.chatID == 0 ||
		op.screener.topicID < 0 || op.simulation.topicID < 0 || op.screener == op.simulation):
		return fmt.Errorf("configure two different Telegram destinations using chat IDs and optional topic IDs")
	case !op.checkFeed && strings.TrimSpace(op.stateDir) == "":
		return fmt.Errorf("state-dir must be set")
	}
	cfg := op.strategy
	for _, value := range []float64{cfg.MinNotional, cfg.MinMultiple, cfg.MaxDistance, cfg.PaperNotional,
		cfg.SpotFee, cfg.LinearFee, cfg.RewardRisk, cfg.MinTarget} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("strategy parameters must be finite")
		}
	}
	if cfg.MinNotional <= 0 || cfg.MinMultiple <= 0 || cfg.MaxDistance <= 0 || cfg.MaxDistance > 1 ||
		cfg.Persistence <= 0 || cfg.Freshness <= 0 {
		return fmt.Errorf("invalid density thresholds or durations")
	}
	if cfg.PaperNotional <= 0 || cfg.SpotFee < 0 || cfg.SpotFee >= 1 || cfg.LinearFee < 0 ||
		cfg.LinearFee >= 1 || cfg.StopTicks <= 0 || cfg.EntryZoneTicks < 0 || cfg.RewardRisk <= 0 ||
		cfg.MinTarget < 0 || cfg.MinTarget >= 1 {
		return fmt.Errorf("invalid simulation sizing, fees or price parameters")
	}
	return nil
}
