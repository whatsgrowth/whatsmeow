package whatsmeow

import (
	"errors"
	"testing"
)

func TestClassifyHistorySyncProcessingErrorDoesNotExposePrivateDetails(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{errors.New("failed to download: private URL"), "download_failed"},
		{errors.New("failed to prepare to decompress: private bytes"), "decompress_prepare_failed"},
		{errors.New("failed to decompress: private bytes"), "decompress_failed"},
		{errors.New("failed to unmarshal: private payload"), "unmarshal_failed"},
		{errors.New("private internal detail"), "processing_failed"},
	}

	for _, test := range tests {
		if got := classifyHistorySyncProcessingError(test.err); got != test.want {
			t.Fatalf("expected %q, got %q", test.want, got)
		}
	}
}
