package bonds

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	"tinvest/internal/enum"
	"tinvest/internal/utils"
)

func (s *service) Trade(ctx context.Context) error {

	bonds, err := s.instrumentServiceGrpcClient.Bonds(ctx)
	if err != nil {
		return err
	}

	for _, bond := range bonds {
		if bond.Exchange != "moex_morning_evening_ofz" || bond.FloatingCouponFlag == true {
			continue
		}
		days := dayDiff(bond.MaturityDate, time.Now())
		limit := int32(10)
		candles, _ := s.marketDataServiceGrpcClient.GetCandles(
			ctx,
			&bond.Id,
			int32(enum.Day1),
			utils.TimeStampPbGenerator(time.Now(), -20, enum.Day1),
			timestamppb.New(time.Now()),
			&limit,
			true,
		)

		fmt.Println(candles)
		if days < 365 {
			continue
		}

		fmt.Println(bond, days)
	}
	fmt.Println(bonds)
	return nil
}

func dayDiff(dateStart time.Time, dateNow time.Time) int {
	currentDate := time.Date(dateNow.Year(), dateNow.Month(), dateNow.Day(), 0, 0, 0, 0, dateNow.Location())
	diff := dateStart.Sub(currentDate)

	return int(diff.Hours() / 24)
}
