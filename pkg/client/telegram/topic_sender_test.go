package telegram_test

import (
	"testing"

	"tinvest/pkg/client/telegram"
	"tinvest/pkg/client/telegram/mocks"
)

func TestTopicSenderBindsSendMessageToTopic(t *testing.T) {
	m := mocks.NewMockClient(t)
	m.EXPECT().SendMessageToTopic(int64(-1001234), 42, "hello").Return(nil)

	s := telegram.NewTopicSender(m, -1001234, 42)
	if err := s.SendMessage("hello"); err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}
}

func TestTopicSenderDelegatesExplicitDestinations(t *testing.T) {
	m := mocks.NewMockClient(t)
	m.EXPECT().SendMessageToChat(int64(7), "a").Return(nil)
	m.EXPECT().SendMessageToTopic(int64(8), 9, "b").Return(nil)

	s := telegram.NewTopicSender(m, -1001234, 42)
	if err := s.SendMessageToChat(7, "a"); err != nil {
		t.Fatalf("SendMessageToChat: %v", err)
	}
	if err := s.SendMessageToTopic(8, 9, "b"); err != nil {
		t.Fatalf("SendMessageToTopic: %v", err)
	}
}
