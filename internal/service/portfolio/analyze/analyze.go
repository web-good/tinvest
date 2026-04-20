package analyze

import (
	"context"
	"fmt"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/utils"
)

func (s service) BondsPortfolio(ctx context.Context) error {
	//level1 := time.Now().String() + "-" + time.Now().AddDate(2, 0, 0).String()
	//level2 := time.Now().AddDate(2, 0, 0).String() + "-" + time.Now().AddDate(6, 0, 0).String()
	//level3 := time.Now().AddDate(6, 0, 0).String() + "-" + time.Now().AddDate(16, 0, 0).String()

	accounts, _ := s.usersServiceClient.GetAccounts(ctx)

	if len(accounts) == 0 {
		return nil
	}

	bondsReport := make(map[string]domain.PortfolioBond)
	bondsReport["level1"] = domain.PortfolioBond{
		Bonds: []*domain.Bond{},
	}

	for _, account := range accounts {
		portfolio, errr := s.operationsServiceClient.GetPortfolio(ctx, account.ID)
		if errr != nil {
			return errr
		}

		if len(portfolio) == 0 {
			continue
		}

		i := 0

		for _, pr := range portfolio {
			i++
			if pr.InstrumentType == "bond" {
				b, err := s.instrumentServiceGrpcClient.BondByID(ctx, pr.ShareID)
				if time.Now().AddDate(2, 0, 0).Before(b.MaturityDate) {
					level := bondsReport["level1"]
					level.Bonds = append(level.Bonds, &domain.Bond{
						Name:     b.Name,
						FinalSum: float64(pr.Quantity) * utils.CombinePrice(pr.Price.Units, pr.Price.Nano),
					})
					bondsReport["level1"] = level
				}

				if err != nil {
					return err
				}
			}
		}
	}

	for _, bof := range bondsReport["level1"].Bonds {
		fmt.Println(bof)
	}
	fmt.Println(bondsReport)
	return nil
}
