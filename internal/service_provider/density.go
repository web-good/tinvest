package service_provider

import (
	"context"
	"fmt"
	"tinvest/internal/service/screener/density"
	internalgrpc "tinvest/pkg/client/grpc"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type densitySender struct {
	bot     *tgbot.Bot
	chatID  int64
	topicID int
	onSent  func(density.Signal)
}

func (s densitySender) Send(ctx context.Context, signal density.Signal) error {
	_, err := s.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: s.chatID, MessageThreadID: s.topicID, Text: density.FormatSignal(signal), ParseMode: models.ParseModeHTML})
	if err == nil && s.onSent != nil {
		s.onSent(signal)
	}
	return err
}
func (s *ServiceProvider) GetDensityService() (*density.Service, error) {
	c := s.appConfig.Density
	if c == nil || !c.Enabled {
		return nil, nil
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("density configuration: %w", err)
	}
	client, err := s.GetGrpcClient()
	if err != nil {
		return nil, err
	}
	bot, err := s.GetTelegramBot()
	if err != nil {
		return nil, err
	}
	market := internalgrpc.NewDensityMarketData(client.Connection(), s.appConfig.GrpcClient.TokenProd)
	cfg := density.DefaultConfig()
	cfg.RequireAverageVolume = true
	cfg.AlertDistanceTicks = c.AlertDistanceTicks
	cfg.AlertM5VolumeMultiplier = c.AlertM5VolumeMultiplier
	cfg.MinRubles = c.MinRubles
	cfg.TurnoverFraction = c.TurnoverFraction
	cfg.Support.Enabled = c.SupportEnabled
	cfg.Support.LookbackDays = c.SupportLookbackDays
	cfg.Support.PivotBars = c.SupportPivotBars
	cfg.Support.MinTouches = c.SupportMinTouches
	cfg.Support.MinSeparation = c.SupportMinSeparation
	cfg.Support.BounceFraction = c.SupportBounceFraction
	cfg.Support.BounceBars = c.SupportBounceBars
	cfg.Support.Tolerance = c.SupportTolerance
	cfg.MinMultiple = c.MinMultiple
	cfg.MinNeighbors = c.MinNeighbors
	cfg.MaxNeighbors = c.MaxNeighbors
	cfg.Persistence = c.Persistence
	cfg.Freshness = c.Freshness
	cfg.MinTurnover = c.MinTurnover
	cfg.MaxLotPrice = c.MaxLotPrice
	cfg.MaxBooks = c.MaxBooks
	cfg.MaxTrades = c.MaxTrades
	cfg.BookDepth = int32(c.BookDepth)
	return density.NewService(cfg, market, market, densitySender{bot: bot.API(), chatID: c.ChatID, topicID: c.TopicID, onSent: s.service.densityAlerts.Record}, c.HistoryInterval), nil
}
