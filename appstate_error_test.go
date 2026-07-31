package whatsmeow

import (
	"errors"
	"fmt"
	"testing"

	"go.mau.fi/whatsmeow/appstate"
	waBinary "go.mau.fi/whatsmeow/binary"
)

func TestNewAppStateUpdateErrorUsesSafeNumericCode(t *testing.T) {
	t.Parallel()

	collection := waBinary.Node{Attrs: waBinary.Attrs{"code": "500"}}
	errorTag := waBinary.Node{Attrs: waBinary.Attrs{
		"code":    "400",
		"private": "must-not-appear",
	}}

	err := newAppStateUpdateError(collection, errorTag, true)
	if err.Code != 400 {
		t.Fatalf("error code = %d, want 400", err.Code)
	}
	if !errors.Is(err, ErrAppStateUpdate) {
		t.Fatal("typed error does not preserve ErrAppStateUpdate compatibility")
	}
	if err.Phase != AppStateUpdatePhaseServerRejected {
		t.Fatalf("error phase = %s, want server_rejected", err.Phase)
	}
	if got := err.Error(); got != "server returned error updating app state (code 400, phase server_rejected)" {
		t.Fatalf("error text = %q, want sanitized code only", got)
	}
}

func TestNewAppStateUpdateErrorFallsBackToCollectionCode(t *testing.T) {
	t.Parallel()

	collection := waBinary.Node{Attrs: waBinary.Attrs{"code": "409"}}
	err := newAppStateUpdateError(collection, waBinary.Node{}, false)

	if err.Code != 409 {
		t.Fatalf("error code = %d, want 409", err.Code)
	}
}

func TestAppStateUpdateErrorWithoutCodeRemainsCompatible(t *testing.T) {
	t.Parallel()

	err := newAppStateUpdateError(waBinary.Node{}, waBinary.Node{}, false)
	if err.Code != 0 {
		t.Fatalf("error code = %d, want 0", err.Code)
	}
	if err.Error() != "server returned error updating app state (phase server_rejected)" {
		t.Fatalf("error text = %q, want safe phase only", err.Error())
	}
}

func TestAppStateUpdatePhaseStringRejectsUnknownValues(t *testing.T) {
	t.Parallel()

	if got := AppStateUpdatePhase(255).String(); got != "unknown" {
		t.Fatalf("unknown phase string = %q, want unknown", got)
	}
}

func TestAppStateUpdateDetailStringRejectsUnknownValues(t *testing.T) {
	t.Parallel()

	if got := AppStateUpdateDetail(255).String(); got != "unknown" {
		t.Fatalf("unknown detail string = %q, want unknown", got)
	}
}

func TestWithAppStateUpdatePhaseAnnotatesOnlyTypedErrors(t *testing.T) {
	t.Parallel()

	original := &AppStateUpdateError{Code: 409, Phase: AppStateUpdatePhaseServerRejected}
	phased := withAppStateUpdatePhase(original, AppStateUpdatePhaseRetryRejected)
	var appStateErr *AppStateUpdateError
	if !errors.As(phased, &appStateErr) || appStateErr.Phase != AppStateUpdatePhaseRetryRejected {
		t.Fatalf("phased error = %#v, want retry_rejected", phased)
	}
	if original.Phase != AppStateUpdatePhaseServerRejected {
		t.Fatal("phase helper mutated the original error")
	}

	plain := errors.New("plain")
	if got := withAppStateUpdatePhase(plain, AppStateUpdatePhaseRetryRejected); got != plain {
		t.Fatal("phase helper replaced an unrelated error")
	}
}

func TestClassifyAppStateApplyFailureUsesOnlyKnownDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		want     AppStateUpdateDetail
		wantText string
	}{
		{name: "LTHash", err: appstate.ErrMismatchingLTHash, want: AppStateUpdateDetailMismatchingLTHash, wantText: "mismatching_lthash"},
		{name: "key", err: appstate.ErrKeyNotFound, want: AppStateUpdateDetailKeyNotFound, wantText: "key_not_found"},
		{name: "patch MAC", err: appstate.ErrMismatchingPatchMAC, want: AppStateUpdateDetailMismatchingPatchMAC, wantText: "mismatching_patch_mac"},
		{name: "content MAC", err: appstate.ErrMismatchingContentMAC, want: AppStateUpdateDetailMismatchingContentMAC, wantText: "mismatching_content_mac"},
		{name: "index MAC", err: appstate.ErrMismatchingIndexMAC, want: AppStateUpdateDetailMismatchingIndexMAC, wantText: "mismatching_index_mac"},
		{name: "previous value", err: appstate.ErrMissingPreviousSetValueOperation, want: AppStateUpdateDetailMissingPreviousValue, wantText: "missing_previous_value"},
		{name: "event collection", err: fmt.Errorf("%w: private", errAppStateEventCollection), want: AppStateUpdateDetailEventCollectionFailed, wantText: "event_collection_failed"},
		{name: "other", err: errors.New("private payload"), want: AppStateUpdateDetailInternalDecodeError, wantText: "internal_decode_error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyAppStateApplyFailure(fmt.Errorf("wrapped: %w", tc.err))
			if got != tc.want || got.String() != tc.wantText {
				t.Fatalf("detail = (%d, %q), want (%d, %q)", got, got.String(), tc.want, tc.wantText)
			}
		})
	}
}
