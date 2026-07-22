package whatsmeow

import (
	"context"
	"testing"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func TestNewClientTracksWhetherDeviceIsAlreadyPaired(t *testing.T) {
	unpaired := NewClient(&store.Device{}, waLog.Noop)
	if unpaired.paired.Load() {
		t.Fatal("new device must not be marked as paired")
	}

	jid := types.NewJID("fixture-device", types.DefaultUserServer)
	paired := NewClient(&store.Device{ID: &jid}, waLog.Noop)
	if !paired.paired.Load() {
		t.Fatal("stored device must be marked as paired")
	}
}

func TestConnectSuccessBeforePairingIsIgnored(t *testing.T) {
	client := NewClient(&store.Device{}, waLog.Noop)
	client.handleConnectSuccess(context.Background(), &waBinary.Node{})

	if client.IsLoggedIn() {
		t.Fatal("connect success before PairSuccess must not authenticate the client")
	}
}

func TestExpectedDisconnectStopsOldQREmitter(t *testing.T) {
	client := NewClient(&store.Device{}, waLog.Noop)
	client.expectDisconnect()
	output := make(chan QRChannelItem, 1)
	channel := qrChannel{
		cli:     client,
		ctx:     context.Background(),
		log:     waLog.Noop,
		output:  output,
		stopQRs: make(chan struct{}),
	}

	channel.emitQRs([]string{"fixture-code"})

	if channel.closed.Load() {
		t.Fatal("expected disconnect must stop the old QR emitter without closing the new client")
	}
}
