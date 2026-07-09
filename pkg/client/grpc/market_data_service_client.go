package grpc

import (
	"context"
	"fmt"
	"time"
	"tinvest/internal/converter"
	"tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"

	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MarketDataServiceClient interface {
	GetTechAnalyseMacD(context context.Context, instrumentUID string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp, fastLength int32) ([]*model.MacDItemTechAnalyse, error)
	GetTechAnalyseRsi(context context.Context, instrumentUID string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp, length int32) ([]*model.RsiItemTechAnalyse, error)
	GetTechAnalyseEma(context context.Context, instrumentUID string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp, length int32) ([]*model.EmaItemTechAnalyse, error)
	GetTechAnalyseBB(context context.Context, instrumentUID string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp) ([]*model.BbItemTechAnalyse, error)
	GetCandles(context context.Context, instrumentUID *string, interval int32, from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withHoliday bool) ([]*model.CandleItemTechAnalyse, error)
	GetTechAnalyseMacDCustom(ctx context.Context, instrumentUID string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp, fastLength int32, slowLength int32, smoothing int32) ([]*model.MacDItemTechAnalyse, error)
}

type marketDataService struct {
	marketDataAPI investapi.MarketDataServiceClient
	auth          *Auth
}

func (m *marketDataService) GetTechAnalyseBB(ctx context.Context, instrumentUID string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp) ([]*model.BbItemTechAnalyse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := m.marketDataAPI.GetTechAnalysis(ctx, &investapi.GetTechAnalysisRequest{
		Length:        20,
		IndicatorType: investapi.GetTechAnalysisRequest_INDICATOR_TYPE_BB,
		InstrumentUid: instrumentUID,
		From:          from,
		To:            to,
		Interval:      investapi.GetTechAnalysisRequest_IndicatorInterval(interval),
		TypeOfPrice:   investapi.GetTechAnalysisRequest_TYPE_OF_PRICE_CLOSE,
		Deviation: &investapi.GetTechAnalysisRequest_Deviation{
			DeviationMultiplier: &investapi.Quotation{Units: 2},
		},
	}, NewRPCCredential(m.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request TechAnalysis: %w", err)
	}

	return converter.ConvertBbTechAnalysisFromPb(resp.GetTechnicalIndicators()), nil
}

func (m *marketDataService) GetTechAnalyseEma(ctx context.Context, instrumentUID string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp, length int32) ([]*model.EmaItemTechAnalyse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := m.marketDataAPI.GetTechAnalysis(ctx, &investapi.GetTechAnalysisRequest{
		Length:        length,
		IndicatorType: investapi.GetTechAnalysisRequest_INDICATOR_TYPE_EMA,
		InstrumentUid: instrumentUID,
		From:          from,
		To:            to,
		Interval:      investapi.GetTechAnalysisRequest_IndicatorInterval(interval),
		TypeOfPrice:   investapi.GetTechAnalysisRequest_TYPE_OF_PRICE_CLOSE,
	}, NewRPCCredential(m.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request TechAnalysis: %w", err)
	}

	return converter.ConvertEmaTechAnalysisFromPb(resp.GetTechnicalIndicators()), nil
}

func (m *marketDataService) GetTechAnalyseRsi(ctx context.Context, instrumentUID string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp, length int32) ([]*model.RsiItemTechAnalyse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := m.marketDataAPI.GetTechAnalysis(ctx, &investapi.GetTechAnalysisRequest{
		Length:        length,
		IndicatorType: investapi.GetTechAnalysisRequest_INDICATOR_TYPE_RSI,
		InstrumentUid: instrumentUID,
		From:          from,
		To:            to,
		Interval:      investapi.GetTechAnalysisRequest_IndicatorInterval(interval),
		TypeOfPrice:   investapi.GetTechAnalysisRequest_TYPE_OF_PRICE_CLOSE,
	}, NewRPCCredential(m.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request TechAnalysis: %w", err)
	}

	return converter.ConvertRsiTechAnalysisFromPb(resp.GetTechnicalIndicators()), nil
}

func (m *marketDataService) GetTechAnalyseMacD(ctx context.Context, instrumentUID string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp, fastLength int32) ([]*model.MacDItemTechAnalyse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := m.marketDataAPI.GetTechAnalysis(ctx, &investapi.GetTechAnalysisRequest{
		IndicatorType: investapi.GetTechAnalysisRequest_INDICATOR_TYPE_MACD,
		InstrumentUid: instrumentUID,
		From:          from,
		To:            to,
		Interval:      investapi.GetTechAnalysisRequest_IndicatorInterval(interval),
		TypeOfPrice:   investapi.GetTechAnalysisRequest_TYPE_OF_PRICE_CLOSE,
		Smoothing: &investapi.GetTechAnalysisRequest_Smoothing{
			FastLength:      fastLength,
			SlowLength:      26,
			SignalSmoothing: 9,
		},
	}, NewRPCCredential(m.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request TechAnalysis: %w", err)
	}

	return converter.ConvertMacDTechAnalysisFromPb(resp.GetTechnicalIndicators()), nil
}

func (m *marketDataService) GetTechAnalyseMacDCustom(ctx context.Context, instrumentUID string, interval int, from *timestamppb.Timestamp, to *timestamppb.Timestamp, fastLength int32, slowLength int32, smoothing int32) ([]*model.MacDItemTechAnalyse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := m.marketDataAPI.GetTechAnalysis(ctx, &investapi.GetTechAnalysisRequest{
		IndicatorType: investapi.GetTechAnalysisRequest_INDICATOR_TYPE_MACD,
		InstrumentUid: instrumentUID,
		From:          from,
		To:            to,
		Interval:      investapi.GetTechAnalysisRequest_IndicatorInterval(interval),
		TypeOfPrice:   investapi.GetTechAnalysisRequest_TYPE_OF_PRICE_CLOSE,
		Smoothing: &investapi.GetTechAnalysisRequest_Smoothing{
			FastLength:      fastLength,
			SlowLength:      slowLength,
			SignalSmoothing: smoothing,
		},
	}, NewRPCCredential(m.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request TechAnalysis: %w", err)
	}

	return converter.ConvertMacDTechAnalysisFromPb(resp.GetTechnicalIndicators()), nil
}

func (m *marketDataService) GetCandles(ctx context.Context, instrumentUID *string, interval int32, from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withHoliday bool) ([]*model.CandleItemTechAnalyse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	h := investapi.GetCandlesRequest_CANDLE_SOURCE_UNSPECIFIED

	if withHoliday {
		h = investapi.GetCandlesRequest_CANDLE_SOURCE_UNSPECIFIED
	}

	resp, err := m.marketDataAPI.GetCandles(ctx, &investapi.GetCandlesRequest{
		From:             from,
		InstrumentId:     instrumentUID,
		To:               to,
		Interval:         investapi.CandleInterval(interval),
		Limit:            limit,
		CandleSourceType: &h,
	}, NewRPCCredential(m.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request TechAnalysis: %w", err)
	}

	return converter.ConvertCandlesTechAnalysisFromPb(resp.GetCandles()), nil
}

func NewMarketDataService(conn grpc.ClientConnInterface, token string) MarketDataServiceClient {
	return &marketDataService{
		marketDataAPI: investapi.NewMarketDataServiceClient(conn),
		auth:          NewAuth(token),
	}
}
