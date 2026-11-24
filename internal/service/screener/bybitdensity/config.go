package bybitdensity

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
	spotProfiles  map[string]spotProfile
	depth         int
	screener      telegramDestination
	simulation    telegramDestination
	telegramToken string
	checkTimeout  time.Duration
	restURL       string
	wsURL         string
}

func readOptions(args []string, out io.Writer, checkFeed bool) (options, error) {
	op := options{strategy: strategy.DefaultConfig(), spotProfiles: defaultSpotProfiles()}
	cfg := &op.strategy
	fs := flag.NewFlagSet("bybit-check-feed", flag.ContinueOnError)
	fs.SetOutput(out)
	envFile := fs.String("env-file", "", "optional dotenv file")
	fs.BoolVar(&cfg.ZoneAlerts, "zone-alerts", true, "alert on persistent bid zones near the market")
	fs.StringVar(&op.stateDir, "state-dir", "reports/bybit-density", "local simulation state directory")
	fs.IntVar(&op.depth, "depth", 200, "Bybit order book depth per side")
	fs.Int64Var(&op.screener.chatID, "screener-chat-id", 0, "Telegram chat for confirmed densities")
	fs.IntVar(&op.screener.topicID, "screener-topic-id", 0, "optional Telegram topic for confirmed densities")
	fs.Int64Var(&op.simulation.chatID, "simulation-chat-id", 0, "Telegram chat for simulated fills")
	fs.IntVar(&op.simulation.topicID, "simulation-topic-id", 0, "optional Telegram topic for simulated fills")
	fs.DurationVar(&op.checkTimeout, "check-timeout", 90*time.Second, "timeout for public feed check")
	fs.StringVar(&op.restURL, "rest-url", "", "override the Bybit REST base URL")
	fs.StringVar(&op.wsURL, "ws-url", "", "override the Bybit WebSocket base URL")
	fs.Int64Var(&cfg.AlertDistanceTicks, "alert-distance-ticks", cfg.AlertDistanceTicks, "maximum distance from best bid to alert on a density, in ticks")
	fs.Float64Var(&cfg.MinMultiple, "min-multiple", cfg.MinMultiple, "minimum multiple of neighbor median")
	fs.DurationVar(&cfg.Persistence, "persistence", cfg.Persistence, "minimum density confirmation time")
	fs.DurationVar(&cfg.Freshness, "freshness", cfg.Freshness, "maximum age of a book update")
	fs.Float64Var(&cfg.PaperNotional, "paper-notional", cfg.PaperNotional, "maximum virtual entry notional in USDT")
	fs.Float64Var(&cfg.SpotFee, "spot-fee", cfg.SpotFee, "simulated spot commission per side as a fraction")
	fs.Int64Var(&cfg.StopTicks, "stop-ticks", cfg.StopTicks, "legacy stop distance (unused)")
	fs.Int64Var(&cfg.EntryTicks, "entry-ticks", cfg.EntryTicks, "legacy entry offset (unused)")
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
		if set[f.Name] || f.Name == "env-file" || f.Name == "stop-ticks" || envErr != nil {
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
	return op, op.validate(checkFeed)
}

func (op options) validate(checkFeed bool) error {
	switch {
	case op.depth != 50 && op.depth != 200 && op.depth != 1000:
		return fmt.Errorf("depth must be 50, 200 or 1000")
	case op.checkTimeout <= 0:
		return fmt.Errorf("check-timeout must be positive")
	case !checkFeed && op.telegramToken == "":
		return fmt.Errorf("set TELEGRAM or %sTELEGRAM_TOKEN", envPrefix)
	case !checkFeed && (op.screener.chatID == 0 || op.simulation.chatID == 0 ||
		op.screener.topicID < 0 || op.simulation.topicID < 0 || op.screener == op.simulation):
		return fmt.Errorf("configure two different Telegram destinations using chat IDs and optional topic IDs")
	case !checkFeed && strings.TrimSpace(op.stateDir) == "":
		return fmt.Errorf("state-dir must be set")
	}
	cfg := op.strategy
	if cfg.AlertDistanceTicks < 0 || cfg.AlertDistanceTicks > 1_000_000 {
		return fmt.Errorf("%sALERT_DISTANCE_TICKS must be between 0 and 1000000", envPrefix)
	}
	for _, value := range []float64{cfg.MinMultiple, cfg.PaperNotional,
		cfg.SpotFee, cfg.RewardRisk, cfg.MinTarget} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("strategy parameters must be finite")
		}
	}
	if cfg.MinMultiple <= 0 ||
		cfg.Persistence <= 0 || cfg.Freshness <= 0 {
		return fmt.Errorf("invalid density thresholds or durations")
	}
	if cfg.PaperNotional <= 0 || cfg.SpotFee < 0 || cfg.SpotFee >= 1 ||
		cfg.StopTicks <= 0 || cfg.EntryTicks < 1 || cfg.EntryTicks > 1000000 || cfg.RewardRisk <= 0 ||
		cfg.MinTarget < 0 || cfg.MinTarget >= 1 {
		return fmt.Errorf("invalid simulation sizing, fees or price parameters")
	}
	return nil
}
