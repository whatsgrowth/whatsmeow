package whatsmeow

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestInvalidateUserDevicesCache(t *testing.T) {
	recipient := types.NewJID("5511999999999", types.DefaultUserServer)
	other := types.NewJID("5511888888888", types.DefaultUserServer)
	client := &Client{
		userDevicesCache: map[types.JID]deviceCache{
			recipient: {},
			other:     {},
		},
	}

	client.invalidateUserDevicesCache(recipient)

	if _, exists := client.userDevicesCache[recipient]; exists {
		t.Fatal("rejected recipient remained in the device cache")
	}
	if _, exists := client.userDevicesCache[other]; !exists {
		t.Fatal("unrelated recipient was removed from the device cache")
	}
}
