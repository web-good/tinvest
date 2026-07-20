package notification

import (
	"strings"
	"testing"
	"time"
	"tinvest/internal/domain"
)

func TestSend_ContainsYTMAndSectorAndPolicy(t *testing.T) {
	bonds := []domain.BondReport{
		{Name: "ОФЗ 26238", PercentByYear: 14.2, CouponPercentByYear: 12.1, Sector: "government", ExecutionDate: time.Now()},
	}
	msg := Send(bonds, time.Now(), time.Now().AddDate(1, 0, 0))

	if !strings.Contains(msg, "YTM") {
		t.Fatal("в сообщении нет метки YTM")
	}
	if !strings.Contains(msg, "government") {
		t.Fatal("в сообщении нет сектора")
	}
	if !strings.Contains(msg, "LOW risk") {
		t.Fatal("в сообщении нет строки политики отбора")
	}
}
