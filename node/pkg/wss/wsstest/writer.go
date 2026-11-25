// Package wsstest provides test helpers for code using package wss.
package wsstest

import (
	"context"
	"encoding/json"
	"sync"

	"bisonai.com/orakl/node/pkg/wss"
)

// Writer is a wss.ConnWriter that records every frame instead of sending it.
// Write frames are recorded as their JSON encoding, RawWrite frames as is.
type Writer struct {
	mu     sync.Mutex
	frames []string
	err    error
}

// SetErr makes every later write return err (nothing is recorded); nil
// restores normal recording.
func (w *Writer) SetErr(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.err = err
}

var _ wss.ConnWriter = (*Writer)(nil)

func (w *Writer) Write(_ context.Context, message any) error {
	b, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return w.record(string(b))
}

func (w *Writer) RawWrite(_ context.Context, message string) error {
	return w.record(message)
}

func (w *Writer) record(frame string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return w.err
	}
	w.frames = append(w.frames, frame)
	return nil
}

// Frames returns the recorded frames in order.
func (w *Writer) Frames() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.frames...)
}
