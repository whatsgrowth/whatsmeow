package whatsmeow

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

func TestStoreHistoricalMessageSecretsStoresPinnedConversationSettings(t *testing.T) {
	ownID := types.NewJID("5511000000000", types.DefaultUserServer)
	chatSettings := &historySyncChatSettingsStore{
		settings: make(map[types.JID]types.LocalChatSettings),
	}
	client := &Client{
		Log: waLog.Noop,
		Store: &store.Device{
			ID:           &ownID,
			ChatSettings: chatSettings,
		},
	}

	client.storeHistoricalMessageSecrets(context.Background(), []*waHistorySync.Conversation{
		{
			ID:     proto.String("5511999999999@s.whatsapp.net"),
			Pinned: proto.Uint32(1),
		},
		{
			ID:     proto.String("5511888888888@s.whatsapp.net"),
			Pinned: proto.Uint32(0),
		},
		{
			ID: proto.String("5511777777777@s.whatsapp.net"),
		},
	})

	pinnedChat := types.NewJID("5511999999999", types.DefaultUserServer)
	if got := chatSettings.settings[pinnedChat]; !got.Found || !got.Pinned {
		t.Fatalf("expected pinned chat setting, got %#v", got)
	}
	unpinnedChat := types.NewJID("5511888888888", types.DefaultUserServer)
	if got := chatSettings.settings[unpinnedChat]; !got.Found || got.Pinned {
		t.Fatalf("expected unpinned chat setting, got %#v", got)
	}
	absentChat := types.NewJID("5511777777777", types.DefaultUserServer)
	if got := chatSettings.settings[absentChat]; got.Found {
		t.Fatalf("did not expect absent pinned field to update chat settings: %#v", got)
	}
}

type historySyncChatSettingsStore struct {
	settings map[types.JID]types.LocalChatSettings
}

func (h *historySyncChatSettingsStore) PutMutedUntil(ctx context.Context, chat types.JID, mutedUntil time.Time) error {
	return nil
}

func (h *historySyncChatSettingsStore) PutPinned(ctx context.Context, chat types.JID, pinned bool) error {
	h.settings[chat] = types.LocalChatSettings{Found: true, Pinned: pinned}
	return nil
}

func (h *historySyncChatSettingsStore) PutArchived(ctx context.Context, chat types.JID, archived bool) error {
	return nil
}

func (h *historySyncChatSettingsStore) GetChatSettings(ctx context.Context, chat types.JID) (types.LocalChatSettings, error) {
	return h.settings[chat], nil
}

func (h *historySyncChatSettingsStore) GetAllChatSettings(ctx context.Context) (map[types.JID]types.LocalChatSettings, error) {
	return h.settings, nil
}
