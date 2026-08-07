package scheduler

import (
	"os"
	"testing"

	"tinvest/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init() // обработчик паники пишет в лог; без Init он сам упал бы на nil
	os.Exit(m.Run())
}

// cron запускает каждое задание в отдельной горутине, поэтому неперехваченная паника в нём
// фатальна для всего процесса: сбой в одном воркере (скажем, дайджесте новостей) убил бы
// живые торговые воркеры, ведущие открытые позиции, и они остались бы мёртвыми до перезапуска
// контейнера. Цена перехвата — один пропущенный запуск: расписание продолжает тикать.
func TestPanickingJobDoesNotEscape(t *testing.T) {
	s := NewScheduler().(*cron)
	if err := s.AddJob("@every 1h", func() { panic("boom") }); err != nil {
		t.Fatalf("AddJob: %v", err)
	}
	entries := s.sh.Entries()
	if len(entries) != 1 {
		t.Fatalf("зарегистрировано заданий: %d, want 1", len(entries))
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("паника вышла за пределы задания: %v", r)
		}
	}()
	entries[0].Job.Run()
}

// Обёртка не должна съедать сам вызов.
func TestNormalJobStillRuns(t *testing.T) {
	s := NewScheduler().(*cron)
	ran := 0
	if err := s.AddJob("@every 1h", func() { ran++ }); err != nil {
		t.Fatalf("AddJob: %v", err)
	}
	s.sh.Entries()[0].Job.Run()
	if ran != 1 {
		t.Fatalf("задание выполнено %d раз, want 1", ran)
	}
}

// Некорректное расписание обязано оставаться ошибкой, а не молча регистрироваться.
func TestInvalidScheduleIsAnError(t *testing.T) {
	s := NewScheduler().(*cron)
	if err := s.AddJob("не расписание", func() {}); err == nil {
		t.Fatal("AddJob принял некорректное расписание")
	}
}
