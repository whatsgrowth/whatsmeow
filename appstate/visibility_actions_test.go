package appstate

import (
	"bytes"
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow/proto/waServerSync"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type fixtureAppStateStore struct {
	store.NoopStore
	keyID     []byte
	key       store.AppStateSyncKey
	valueMACs map[string][]byte
	version   uint64
	stateHash [128]byte
}

func newFixtureAppStateStore() *fixtureAppStateStore {
	return &fixtureAppStateStore{
		NoopStore: store.NoopStore{},
		keyID:     []byte("fixture-key-id"),
		key:       store.AppStateSyncKey{Data: bytes.Repeat([]byte{0x42}, 32)},
		valueMACs: make(map[string][]byte),
	}
}

func (s *fixtureAppStateStore) GetAppStateSyncKey(_ context.Context, id []byte) (*store.AppStateSyncKey, error) {
	if !bytes.Equal(id, s.keyID) {
		return nil, nil
	}
	key := s.key
	return &key, nil
}

func (s *fixtureAppStateStore) GetLatestAppStateSyncKeyID(context.Context) ([]byte, error) {
	return append([]byte(nil), s.keyID...), nil
}

func (s *fixtureAppStateStore) PutAppStateVersion(_ context.Context, _ string, version uint64, hash [128]byte) error {
	s.version = version
	s.stateHash = hash
	return nil
}

func (s *fixtureAppStateStore) GetAppStateVersion(context.Context, string) (uint64, [128]byte, error) {
	return s.version, s.stateHash, nil
}

func (s *fixtureAppStateStore) PutAppStateMutationMACs(_ context.Context, _ string, _ uint64, mutations []store.AppStateMutationMAC) error {
	for _, mutation := range mutations {
		s.valueMACs[string(mutation.IndexMAC)] = append([]byte(nil), mutation.ValueMAC...)
	}
	return nil
}

func (s *fixtureAppStateStore) DeleteAppStateMutationMACs(_ context.Context, _ string, indexMACs [][]byte) error {
	for _, indexMAC := range indexMACs {
		delete(s.valueMACs, string(indexMAC))
	}
	return nil
}

func (s *fixtureAppStateStore) GetAppStateMutationMAC(_ context.Context, _ string, indexMAC []byte) ([]byte, error) {
	return append([]byte(nil), s.valueMACs[string(indexMAC)]...), nil
}

func TestVisibilityActionsEncodeDecodeInRegularHigh(t *testing.T) {
	t.Parallel()

	target := types.NewJID("fixture-chat", types.DefaultUserServer)
	cutoff := time.Unix(1_700_000_000, 0).UTC()
	tests := []struct {
		name          string
		patch         PatchInfo
		expectedIndex []string
		isClear       bool
	}{
		{
			name:          "delete chat without media deletion",
			patch:         BuildDeleteChat(target, cutoff, nil, false),
			expectedIndex: []string{IndexDeleteChat, target.String(), "0"},
		},
		{
			name:          "delete chat with media deletion",
			patch:         BuildDeleteChat(target, cutoff, nil, true),
			expectedIndex: []string{IndexDeleteChat, target.String(), "1"},
		},
		{
			name:          "clear chat keeping starred messages without media deletion",
			patch:         BuildClearChat(target, cutoff, nil, true, false),
			expectedIndex: []string{IndexClearChat, target.String(), "0", "0"},
			isClear:       true,
		},
		{
			name:          "clear chat deleting starred messages and media",
			patch:         BuildClearChat(target, cutoff, nil, false, true),
			expectedIndex: []string{IndexClearChat, target.String(), "1", "1"},
			isClear:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.patch.Type != WAPatchRegularHigh {
				t.Fatalf("visibility action encoded in %q, want %q", tc.patch.Type, WAPatchRegularHigh)
			}

			fixtureStore := newFixtureAppStateStore()
			device := &store.Device{AppStateKeys: fixtureStore, AppState: fixtureStore}
			processor := NewProcessor(device, waLog.Noop)
			encoded, err := processor.EncodePatch(context.Background(), fixtureStore.keyID, HashState{}, tc.patch)
			if err != nil {
				t.Fatalf("EncodePatch failed: %v", err)
			}

			var wirePatch waServerSync.SyncdPatch
			if err = proto.Unmarshal(encoded, &wirePatch); err != nil {
				t.Fatalf("unmarshal encoded patch: %v", err)
			}
			wirePatch.Version = &waServerSync.SyncdVersion{Version: proto.Uint64(1)}
			mutations, terminal, err := processor.DecodePatches(context.Background(), &PatchList{
				Name:    tc.patch.Type,
				Patches: []*waServerSync.SyncdPatch{&wirePatch},
			}, HashState{}, true)
			if err != nil {
				t.Fatalf("DecodePatches failed: %v", err)
			}
			if terminal.Version != 1 {
				t.Fatalf("terminal version = %d, want 1", terminal.Version)
			}
			if len(mutations) != 1 {
				t.Fatalf("decoded mutations = %d, want 1", len(mutations))
			}
			if got := mutations[0].Index; !equalStrings(got, tc.expectedIndex) {
				t.Fatalf("decoded index = %v, want %v", got, tc.expectedIndex)
			}
			if mutations[0].Version != 6 {
				t.Fatalf("action version = %d, want 6", mutations[0].Version)
			}
			if tc.isClear {
				if got := mutations[0].Action.GetClearChatAction().GetMessageRange().GetLastMessageTimestamp(); got != cutoff.Unix() {
					t.Fatalf("clear cutoff = %d, want %d", got, cutoff.Unix())
				}
			} else if got := mutations[0].Action.GetDeleteChatAction().GetMessageRange().GetLastMessageTimestamp(); got != cutoff.Unix() {
				t.Fatalf("delete cutoff = %d, want %d", got, cutoff.Unix())
			}
		})
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
