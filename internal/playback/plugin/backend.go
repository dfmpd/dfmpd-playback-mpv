package plugin

import (
	"context"
	"errors"

	"github.com/dfmpd/dfmpd-playback-mpv/internal/playback"
	sdk "github.com/dfmpd/dfmpd-sdk/playback"
)

// NewBackend converts plugin-local adapter values at the SDK boundary.
func NewBackend(a playback.Adapter) sdk.Backend { return &backend{a} }

type backend struct{ adapter playback.Adapter }

func toSDKError(err error) error {
	var e *playback.Error
	if errors.As(err, &e) {
		return &sdk.Error{Code: sdk.Code(e.Code), Message: e.Message}
	}
	return err
}
func (b *backend) Play(ctx context.Context, input sdk.Input) error {
	return toSDKError(b.adapter.Play(ctx, playback.Input{Location: input.Location}))
}
func (b *backend) Pause(ctx context.Context) error  { return toSDKError(b.adapter.Pause(ctx)) }
func (b *backend) Resume(ctx context.Context) error { return toSDKError(b.adapter.Resume(ctx)) }
func (b *backend) Stop(ctx context.Context) error   { return toSDKError(b.adapter.Stop(ctx)) }
func (b *backend) Seek(ctx context.Context, seconds float64) error {
	return toSDKError(b.adapter.Seek(ctx, seconds))
}
func (b *backend) Status(ctx context.Context) (sdk.Status, error) {
	s, err := b.adapter.Status(ctx)
	var failure *sdk.Error
	if s.Failure != nil {
		failure = &sdk.Error{Code: sdk.Code(s.Failure.Code), Message: s.Failure.Message}
	}
	return sdk.Status{State: sdk.State(s.State), Idle: s.Idle, Paused: s.Paused, Position: s.Position, Failure: failure}, toSDKError(err)
}
func (b *backend) Close() error { return toSDKError(b.adapter.Close()) }
