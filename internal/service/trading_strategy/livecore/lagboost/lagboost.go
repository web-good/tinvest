// Package lagboost решает, увеличить ли долю входа по тикеру, который за последние
// Days завершённых дней отстал от индекса IMOEX на Pct п.п. и больше. Пакет чистый:
// ряды закрытий собирает раннер (без сегодняшней незавершённой свечи), здесь — только
// арифметика. Обоснование и пороги — docs/superpowers/specs/2026-10-08-lag-boost-design.md.
package lagboost

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Config — настройки LAG_BOOST_* (общие для раннера счёта rsi_pullback).
type Config struct {
	Enabled bool
	Mult    float64 // множитель доли входа
	Pct     float64 // порог отставания, п.п.
	Days    int     // окно в завершённых днях
}

// Validate проверяет границы только у включённой опции: выключенная с любыми числами
// безопасна. Верхняя граница множителя — защита от опечатки в ENV, а не торговое решение.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	var errs []error
	if c.Mult < 1 || c.Mult > 3 {
		errs = append(errs, fmt.Errorf("LAG_BOOST_MULT=%v вне [1, 3]", c.Mult))
	}
	if c.Pct <= 0 {
		errs = append(errs, fmt.Errorf("LAG_BOOST_PCT=%v должен быть > 0", c.Pct))
	}
	if c.Days < 1 || c.Days > 60 {
		errs = append(errs, fmt.Errorf("LAG_BOOST_DAYS=%d вне [1, 60]", c.Days))
	}
	return errors.Join(errs...)
}

// Decision — итог для одного входа. Mult всегда ≥ 1; Applied — сработал ли буст.
type Decision struct {
	Mult    float64
	Lag     float64 // доходность тикера минус доходность индекса за окно, п.п.
	Applied bool
	Reason  string // почему буст не применён (для лога); пусто, если применён
}

// Decide сравнивает доходности за последние Days+1 закрытий каждого ряда (сдвиг считается
// по барам своего ряда). Любая нехватка данных даёт множитель 1: буст необязателен.
func Decide(tickerCloses, indexCloses []float64, c Config) Decision {
	if !c.Enabled {
		return Decision{Mult: 1, Reason: "выключено"}
	}
	n := c.Days
	if len(tickerCloses) < n+1 || len(indexCloses) < n+1 {
		return Decision{Mult: 1, Reason: fmt.Sprintf("мало истории: тикер %d, индекс %d закрытий, нужно %d",
			len(tickerCloses), len(indexCloses), n+1)}
	}
	t0, t1 := tickerCloses[len(tickerCloses)-1-n], tickerCloses[len(tickerCloses)-1]
	x0, x1 := indexCloses[len(indexCloses)-1-n], indexCloses[len(indexCloses)-1]
	if t0 <= 0 || t1 <= 0 || x0 <= 0 || x1 <= 0 {
		return Decision{Mult: 1, Reason: "нулевая или отрицательная цена в окне"}
	}
	lag := (t1/t0-1)*100 - (x1/x0-1)*100
	if lag <= -c.Pct {
		return Decision{Mult: c.Mult, Lag: lag, Applied: true}
	}
	return Decision{Mult: 1, Lag: lag, Reason: fmt.Sprintf("отставание %.2f п.п. не дотягивает до %v", -lag, c.Pct)}
}

// msk — календарь MOEX: выходной определяется по дню недели в Москве.
var msk = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.FixedZone("MSK", 3*3600)
	}
	return loc
}()

// IsWeekend — суббота или воскресенье по MSK. API отдаёт дневки выходных сессий и у акций,
// и у IMOEX; исследование считало окно в будних днях (ряд индекса с ISS без выходных),
// поэтому оба ряда перед Decide чистятся от выходных — иначе окно Days баров покрывает
// 3–4 торговых дня вместо Days.
func IsWeekend(t time.Time) bool {
	wd := t.In(msk).Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// WeekdayCloses оставляет закрытия будних дней. times выровнены с closes индекс в индекс;
// при несовпадении длин возвращает nil — Decide ответит «мало истории», буст не сработает.
func WeekdayCloses(closes []float64, times []time.Time) []float64 {
	if len(closes) != len(times) {
		return nil
	}
	out := make([]float64, 0, len(closes))
	for i, c := range closes {
		if !IsWeekend(times[i]) {
			out = append(out, c)
		}
	}
	return out
}

// Note — строка для уведомления о входе; пустая, если буст не применён.
func (d Decision) Note(days int) string {
	if !d.Applied {
		return ""
	}
	return fmt.Sprintf("Размер ×%s: отстал от IMOEX на %.1f%% за %dд",
		strconv.FormatFloat(d.Mult, 'f', -1, 64), -d.Lag, days)
}
