package news

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"tinvest/pkg/client/rss"
	rssmocks "tinvest/pkg/client/rss/mocks"
	tgmocks "tinvest/pkg/client/telegram/mocks"
	"tinvest/pkg/logger"
)

// TestMain инициализирует пакетный логгер: Run зовёт logger.InfoContext,
// который паникует на nil-логгере (прецедент — telegram_commands).
func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

var baseTime = time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

func item(guid string, pub time.Time) rss.Item {
	return rss.Item{Title: "n " + guid, Link: "https://example.com/" + guid, GUID: guid, PubDate: pub}
}

// newTestService фиксирует "сейчас" = baseTime: окно первого запуска —
// [baseTime-1h, ...].
func newTestService(f rss.Fetcher, tg *tgmocks.MockClient) *Service {
	s := NewService(f, tg)
	s.now = func() time.Time { return baseTime }
	return s
}

func TestRun_FirstRunPostsOnlyLastHour(t *testing.T) {
	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{
		item("fresh", baseTime.Add(-10*time.Minute)),
		item("stale", baseTime.Add(-2*time.Hour)),
	}, nil).Once()

	tg := tgmocks.NewMockClient(t)
	var sent string
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()

	if err := newTestService(f, tg).Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(sent, "fresh") || strings.Contains(sent, "stale") {
		t.Errorf("в дайджесте только свежие записи, got %q", sent)
	}
}

func TestRun_SecondRunSkipsAlreadySent(t *testing.T) {
	first := item("a", baseTime.Add(-10*time.Minute))
	second := item("b", baseTime.Add(-5*time.Minute))

	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{first}, nil).Once()
	tg := tgmocks.NewMockClient(t)
	var sent string
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()

	svc := newTestService(f, tg)
	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run 1: %v", err)
	}

	// Второй запуск: лента отдаёт old+new — уйти должна только new.
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{second, first}, nil).Once()
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()
	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if strings.Contains(sent, `>n a<`) || !strings.Contains(sent, `>n b<`) {
		t.Errorf("повтор уже отправленной записи, got %q", sent)
	}
}

func TestRun_BoundaryGUIDDedup(t *testing.T) {
	// Две записи с одинаковым PubDate приходят в разных итерациях: вторая
	// не должна потеряться (PubDate == lastSeen) и первая не должна
	// повториться (GUID уже отправлен).
	pub := baseTime.Add(-10 * time.Minute)
	a, b := item("a", pub), item("b", pub)

	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{a}, nil).Once()
	tg := tgmocks.NewMockClient(t)
	var sent string
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()

	svc := newTestService(f, tg)
	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run 1: %v", err)
	}

	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{b, a}, nil).Once()
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()
	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if strings.Contains(sent, `>n a<`) || !strings.Contains(sent, `>n b<`) {
		t.Errorf("граница по GUID отработала неверно, got %q", sent)
	}
}

func TestRun_ChronologicalOrder(t *testing.T) {
	// RSS отдаёт новые сверху — в дайджесте порядок хронологический.
	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{
		item("newer", baseTime.Add(-5*time.Minute)),
		item("older", baseTime.Add(-30*time.Minute)),
	}, nil).Once()
	tg := tgmocks.NewMockClient(t)
	var sent string
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()

	if err := newTestService(f, tg).Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.Index(sent, "older") > strings.Index(sent, "newer") {
		t.Errorf("ожидался хронологический порядок, got %q", sent)
	}
}

func TestRun_NoNewItems_NoSend(t *testing.T) {
	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{
		item("stale", baseTime.Add(-2*time.Hour)),
	}, nil).Once()
	tg := tgmocks.NewMockClient(t) // без EXPECT: любой SendMessage завалит тест

	if err := newTestService(f, tg).Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRun_FetchError_ReturnsError(t *testing.T) {
	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return(nil, errors.New("boom")).Once()
	tg := tgmocks.NewMockClient(t)

	if err := newTestService(f, tg).Run(context.Background()); err == nil {
		t.Fatal("ожидалась ошибка fetch")
	}
}

func TestRun_SendError_DoesNotAdvanceWindow(t *testing.T) {
	it := item("a", baseTime.Add(-10*time.Minute))
	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{it}, nil).Twice()

	tg := tgmocks.NewMockClient(t)
	var sent string
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(errors.New("tg down")).Once()

	svc := newTestService(f, tg)
	if err := svc.Run(context.Background()); err == nil {
		t.Fatal("ожидалась ошибка отправки")
	}

	// Повторный запуск шлёт ту же запись заново — окно не сдвинулось.
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()
	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if !strings.Contains(sent, `>n a<`) {
		t.Errorf("запись потеряна после ошибки отправки, got %q", sent)
	}
}

// TestRun_ConcurrentRuns гоняет два Run параллельно на одном Service: cron не
// гарантирует отсутствие перекрытия тиков, поэтому доступ к состоянию
// (lastSeen, boundaryGUIDs) должен быть сериализован. Запускать с -race.
func TestRun_ConcurrentRuns(t *testing.T) {
	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{
		item("c1", baseTime.Add(-10*time.Minute)),
		item("c2", baseTime.Add(-5*time.Minute)),
	}, nil).Times(2)

	tg := tgmocks.NewMockClient(t)
	tg.EXPECT().SendMessage(mock.AnythingOfType("string")).Return(nil).Maybe()

	svc := newTestService(f, tg)

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			<-start
			if err := svc.Run(context.Background()); err != nil {
				t.Errorf("Run: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
}

// TestRun_FutureDatedItemDoesNotPermanentlyMuteWindow воспроизводит баг:
// без клампа item с PubDate в будущем (перекос часов источника) поднимает
// lastSeen так высоко, что все последующие реальные записи навсегда
// перестают проходить фильтр PubDate.After(lastSeen) — дайджест молчит.
func TestRun_FutureDatedItemDoesNotPermanentlyMuteWindow(t *testing.T) {
	future := item("future", baseTime.Add(24*time.Hour))

	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{future}, nil).Once()
	tg := tgmocks.NewMockClient(t)
	var sent string
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()

	svc := newTestService(f, tg)
	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run 1: %v", err)
	}
	if !strings.Contains(sent, `>n future<`) {
		t.Fatalf("future-item должен уйти в первом запуске, got %q", sent)
	}

	// Следующий тик (час спустя): в ленте обычная свежая запись. Без клампа
	// lastSeen застрял бы на baseTime+24h и эта запись никогда бы не прошла
	// PubDate.After(lastSeen).
	svc.now = func() time.Time { return baseTime.Add(time.Hour) }
	fresh := item("fresh", baseTime.Add(30*time.Minute))
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{fresh}, nil).Once()
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()

	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if !strings.Contains(sent, `>n fresh<`) {
		t.Errorf("окно навсегда заглушено будущей записью, got %q", sent)
	}
}

// TestRun_IntraBatchDuplicateGUID: одна и та же запись пришла в одном ответе
// Fetch дважды — в дайджесте она должна появиться один раз.
func TestRun_IntraBatchDuplicateGUID(t *testing.T) {
	dup := item("dup", baseTime.Add(-10*time.Minute))

	f := rssmocks.NewMockFetcher(t)
	f.EXPECT().Fetch(context.Background()).Return([]rss.Item{dup, dup}, nil).Once()
	tg := tgmocks.NewMockClient(t)
	var sent string
	tg.EXPECT().SendMessage(mockAnyString(&sent)).Return(nil).Once()

	if err := newTestService(f, tg).Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := strings.Count(sent, `>n dup<`); got != 1 {
		t.Errorf("ожидался ровно один пункт dup в дайджесте, встречается %d раз: %q", got, sent)
	}
}

// mockAnyString принимает любое строковое сообщение и запоминает последнее
// в *dst — так проверяем содержимое дайджеста.
func mockAnyString(dst *string) any {
	return mock.MatchedBy(func(s string) bool {
		*dst = s
		return true
	})
}
