package plugin

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/dfmpd/dfmpd-playback-mpv/internal/playback"
	sdk "github.com/dfmpd/dfmpd-sdk/playback"
)

type recordingAdapter struct {
	playback.Adapter
	input   playback.Input
	seconds float64
	calls   []string
}

func (a *recordingAdapter) Play(_ context.Context, in playback.Input) error {
	a.input = in
	a.calls = append(a.calls, "play")
	return nil
}
func (a *recordingAdapter) Pause(context.Context) error {
	a.calls = append(a.calls, "pause")
	return nil
}
func (a *recordingAdapter) Resume(context.Context) error {
	a.calls = append(a.calls, "resume")
	return nil
}
func (a *recordingAdapter) Stop(context.Context) error {
	a.calls = append(a.calls, "stop")
	return &playback.Error{Code: playback.Rejected, Message: "stop rejected"}
}
func (a *recordingAdapter) Seek(_ context.Context, seconds float64) error {
	a.seconds = seconds
	a.calls = append(a.calls, "seek")
	return nil
}
func (a *recordingAdapter) Status(context.Context) (playback.Status, error) {
	a.calls = append(a.calls, "status")
	return playback.Status{State: playback.Failed, Failure: &playback.Error{Code: playback.MediaFailed, Message: "decode failed"}}, nil
}
func (a *recordingAdapter) Close() error { a.calls = append(a.calls, "close"); return nil }
func TestBackendBoundary(t *testing.T) {
	a := &recordingAdapter{}
	b := NewBackend(a)
	ctx := context.Background()
	if err := b.Play(ctx, sdk.Input{Location: "/track.flac"}); err != nil {
		t.Fatal(err)
	}
	if err := b.Pause(ctx); err != nil {
		t.Fatal(err)
	}
	if err := b.Resume(ctx); err != nil {
		t.Fatal(err)
	}
	if err := b.Seek(ctx, 4.5); err != nil {
		t.Fatal(err)
	}
	s, err := b.Status(ctx)
	if err != nil || s.State != sdk.Failed || s.Failure == nil || s.Failure.Code != sdk.MediaFailed {
		t.Fatalf("status=%+v err=%v", s, err)
	}
	if err := b.Stop(ctx); !errors.Is(err, &sdk.Error{Code: sdk.Rejected}) {
		t.Fatalf("stop=%v", err)
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if a.input.Location != "/track.flac" || a.seconds != 4.5 || !reflect.DeepEqual(a.calls, []string{"play", "pause", "resume", "seek", "status", "stop", "close"}) {
		t.Fatalf("calls=%+v", a)
	}
	for _, code := range []playback.Code{playback.InvalidInput, playback.Unavailable, playback.Rejected, playback.TimedOut, playback.Canceled, playback.Closed, playback.MediaFailed} {
		err := toSDKError(&playback.Error{Code: code, Message: "diagnostic"})
		var wire *sdk.Error
		if !errors.As(err, &wire) || wire.Code != sdk.Code(code) || wire.Message != "diagnostic" {
			t.Fatalf("conversion=%v", err)
		}
	}
	if !errors.Is(toSDKError(context.Canceled), context.Canceled) || !errors.Is(toSDKError(&playback.Error{Code: playback.TimedOut}), context.DeadlineExceeded) {
		t.Fatal("context semantics lost")
	}
}
