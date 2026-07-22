package whatsmeow

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type mediaRoundTripperFunc func(*http.Request) (*http.Response, error)

func (fn mediaRoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestDownloadFallsBackToDirectPathWhenPrimaryURLIsUnavailable(t *testing.T) {
	const expected = "sticker-data"
	var primaryAttempts int
	var directPathAttempts int
	client := &Client{
		Log: waLog.Noop,
		mediaHTTP: &http.Client{Transport: mediaRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Host {
			case "primary.invalid":
				primaryAttempts++
				return nil, &net.DNSError{Err: "host unavailable", Name: "redacted.invalid"}
			case "media.invalid":
				directPathAttempts++
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(expected)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			default:
				t.Fatalf("unexpected media host: %s", req.URL.Host)
				return nil, nil
			}
		})},
		mediaConnCache: &MediaConn{
			TTL:       60,
			FetchedAt: time.Now(),
			Hosts:     []MediaConnHost{{Hostname: "media.invalid"}},
		},
	}

	data, err := client.Download(context.Background(), &waE2E.StickerMessage{
		URL:        proto.String("https://primary.invalid/sticker"),
		DirectPath: proto.String("/sticker?token=redacted"),
	})
	if err != nil {
		t.Fatalf("expected direct path fallback to succeed, got %v", err)
	}
	if string(data) != expected {
		t.Fatalf("expected %q, got %q", expected, data)
	}
	if primaryAttempts != 1 {
		t.Fatalf("expected one primary URL attempt before fallback, got %d", primaryAttempts)
	}
	if directPathAttempts != 1 {
		t.Fatalf("expected one direct path attempt, got %d", directPathAttempts)
	}
}

func TestShouldFallbackMediaURLToDirectPath(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "dns failure", err: &net.DNSError{Err: "host unavailable", Name: "redacted.invalid"}, want: true},
		{name: "forbidden", err: ErrMediaDownloadFailedWith403, want: true},
		{name: "not found", err: ErrMediaDownloadFailedWith404, want: true},
		{name: "gone", err: ErrMediaDownloadFailedWith410, want: true},
		{name: "cancelled", err: context.Canceled, want: false},
		{name: "length mismatch", err: ErrFileLengthMismatch, want: false},
		{name: "invalid hash", err: ErrInvalidMediaSHA256, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldFallbackMediaURLToDirectPath(test.err); got != test.want {
				t.Fatalf("expected %t, got %t", test.want, got)
			}
		})
	}
}
