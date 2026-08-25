package whatsmeow

import (
	"encoding/json"
	"testing"
)

func TestReachoutTimelockMexPayloadIsRecognized(t *testing.T) {
	var wrapper newsLetterEventWrapper
	err := json.Unmarshal([]byte(`{
		"data": {
			"xwa2_notify_account_reachout_timelock": {
				"enforcement_type": "RESTRICT_ALL_COMPANIONS",
				"is_active": true,
				"time_enforcement_ends": "1787655600"
			}
		}
	}`), &wrapper)
	if err != nil {
		t.Fatalf("failed to decode reachout timelock payload: %v", err)
	}

	event := wrapper.Data.NotifyAccountReachoutTimelock
	if event == nil {
		t.Fatal("reachout timelock event was not recognized")
	}
	if !event.IsActive {
		t.Fatal("reachout timelock event must preserve active state")
	}
	if event.TimeEnforcementEnds.IsZero() {
		t.Fatal("reachout timelock event must preserve the end time")
	}
}
