package down_pump

import (
	"context"
	"fmt"
	"tinvest/internal/service/trading_strategy/down_pump/dto"
)

func (s *service) Trade(ctx context.Context, in dto.Trade) error {
	fmt.Println(in)
	//info := domain.NewInfo()
	//	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)
	//for _, share := range t {
	//ema, errEma := s.ema.TechAnalyse(ctx, &share.ID, int32(in.Interval), utils.TimeGenerator(dateNow, -350, in.Interval), utils.TimeGenerator(dateNow, -1, in.Interval), 200)

	/*if errEma != nil {
		logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema :%w", errEma).Error())

		return domain.Item{}, errEma
	}*/
	//}
	return nil
}
