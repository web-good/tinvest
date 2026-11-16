package service_provider

import (
	"context"
	"fmt"
	"path/filepath"
	"tinvest/internal/config"
	"tinvest/internal/service/screener/density"
	densitylive "tinvest/internal/service/screener/density/live"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	marketgrpc "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"

	tgbot "github.com/go-telegram/bot"
)

type densityLiveSender struct {
	bot     *tgbot.Bot
	chatID  int64
	topicID int
}

func (s densityLiveSender) Send(ctx context.Context, n densitylive.Notification) error {
	return s.SendText(ctx, densitylive.FormatNotification(n))
}

func (s densityLiveSender) SendText(ctx context.Context, text string) error {
	_, err := s.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: s.chatID, MessageThreadID: s.topicID, Text: text})
	return err
}

func densityLiveAccount(cfg *config.Config) (string, string, error) {
	if cfg == nil || cfg.RSIPullback == nil || !cfg.RSIPullback.Ready() {
		return "", "", fmt.Errorf("density shared account requires RSI_PULLBACK_ACCOUNT_ID and RSI_PULLBACK_TOKEN")
	}
	return cfg.RSIPullback.AccountID, cfg.RSIPullback.Token, nil
}

func (s *ServiceProvider) GetAccountCoordinator() *accountcoord.Coordinator {
	if s.service.accountCoordinator == nil {
		account := s.appConfig.RSIPullback.AccountID
		legacy := statestore.New(filepath.Join("data", "state", "rsi_pullback_"+account+".json"))
		s.service.accountCoordinator = accountcoord.New(filepath.Join("data", "state", "accountcoord"), account, legacy)
		stateDir := config.NewDensityLiveConfig().StateDir
		if c := s.appConfig.DensityLive; c != nil && c.StateDir != "" {
			stateDir = c.StateDir
		}
		coordinator := s.service.accountCoordinator
		coordinator.SetDensitySnapshot(func(context.Context) (map[string]accountcoord.DensityPosition, error) {
			return densitylive.ReadPositions(filepath.Join(stateDir, densitylive.AccountHash(account)))
		})
		coordinator.SetGuard(func() error {
			return densitylive.CheckSharedBinding(filepath.Join(stateDir, densitylive.AccountHash(account)), account, coordinator.Root())
		})
	}
	return s.service.accountCoordinator
}

func (s *ServiceProvider) GetDensityLiveService() (*densitylive.Participant, error) {
	if s.appConfig.DensityLiveError != nil {
		return nil, s.appConfig.DensityLiveError
	}
	c := s.appConfig.DensityLive
	if c == nil || !c.Enabled {
		return nil, nil
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	d := s.appConfig.Density
	if d == nil {
		d = config.NewDensityConfig()
	}
	if d.ChatID == 0 {
		return nil, fmt.Errorf("DENSITY_CHAT_ID is required when DENSITY_LIVE_ENABLED is true")
	}
	if d.TopicID < 0 {
		return nil, fmt.Errorf("DENSITY_TOPIC_ID must be nonnegative when DENSITY_LIVE_ENABLED is true")
	}
	copy := *d
	copy.Enabled = false
	if err := copy.Validate(); err != nil {
		return nil, err
	}
	detector := density.DefaultConfig()
	detector.MinRubles = d.MinRubles
	detector.TurnoverFraction = d.TurnoverFraction
	detector.Support.Enabled = d.SupportEnabled
	detector.Support.LookbackDays = d.SupportLookbackDays
	detector.Support.PivotBars = d.SupportPivotBars
	detector.Support.MinTouches = d.SupportMinTouches
	detector.Support.MinSeparation = d.SupportMinSeparation
	detector.Support.BounceFraction = d.SupportBounceFraction
	detector.Support.BounceBars = d.SupportBounceBars
	detector.Support.Tolerance = d.SupportTolerance
	detector.MinMultiple = d.MinMultiple
	detector.MaxDistance = d.MaxDistance
	detector.MinNeighbors = d.MinNeighbors
	detector.MaxNeighbors = d.MaxNeighbors
	detector.Persistence = d.Persistence
	detector.Freshness = d.Freshness
	detector.MinTurnover = d.MinTurnover
	detector.MaxLotPrice = d.MaxLotPrice
	detector.MaxBooks = min(len(c.Tickers), d.MaxBooks)
	detector.BookDepth = int32(d.BookDepth)
	lc := densitylive.DefaultConfig()
	lc.EntryTicks = c.EntryTicks
	lc.StopTicks = c.StopTicks
	lc.RewardRisk = c.RewardRisk
	lc.MinRemaining = c.MinRemaining
	lc.Freshness = d.Freshness
	lc.EntryTTL = c.EntryTTL
	lc.MaxHold = c.MaxHold
	lc.Cooldown = c.Cooldown
	if !c.TradeEnabled {
		return s.densitySimulationParticipant(c, d, detector, lc), nil
	}
	account, token, err := densityLiveAccount(s.appConfig)
	if err != nil {
		return nil, err
	}
	client, err := s.GetRSIPullbackGrpcClient()
	if err != nil {
		return nil, err
	}
	p := &densitylive.Participant{Account: account, StateDir: c.StateDir, Tickers: c.Tickers, TradeEnabled: c.TradeEnabled, Config: lc, Detector: detector, HistoryInterval: d.HistoryInterval, Market: marketgrpc.NewDensityMarketData(client.Connection(), token), Coordinator: s.GetAccountCoordinator()}
	if c.TradeEnabled {
		bot, botErr := s.GetTelegramBot()
		if botErr != nil {
			return nil, botErr
		}
		p.Notify = densityLiveSender{bot: bot.API(), chatID: d.ChatID, topicID: d.TopicID}.Send
	}
	p.NewBroker = func(instruments []density.Instrument) densitylive.Broker {
		return marketgrpc.NewDensityBroker(client.Connection(), token, account, c.TradeEnabled, instruments)
	}
	p.Warn = func(err error) { logger.ErrorContext(context.Background(), "density shared account: "+err.Error()) }
	p.Report = func(state densitylive.State) error {
		for _, ep := range state.Episodes {
			status := ep.Reason
			if status == "" {
				status = "active"
			}
			if ep.Blocked != "" {
				status = "UNRESOLVED: " + ep.Blocked
			}
			logger.Info(fmt.Sprintf("density live: ticker=%s status=%s remaining_lots=%d", ep.Instrument.Ticker, status, ep.Remaining()))
		}
		logger.Info(fmt.Sprintf("density live: active=%d closed=%d realized=%+.2f RUB pnl_complete=%t", len(state.Episodes), state.Closed, float64(state.Realized)/1e9, state.PnLComplete))
		return nil
	}
	return p, nil
}
