package down_pump

import (
	"context"
	"fmt"
	"tinvest/internal/service/trading_strategy/down_pump/dto"
)

func (s *service) Trade(ctx context.Context, in dto.Trade) error {
	fmt.Println(in)
	//info := domain.NewInfo()
	//t, _ := s.instrumentServiceGrpcClient.Shares(ctx)
	//for _, share := range t {
	//}
	return nil
}
