// Package wsstest provides test helpers for code using package wss.
package wsstest

import (
	"context"
	"encoding/json"
	"sync"

	"bisonai.com/miko/node/pkg/wss"
)

// Writer is a wss.ConnWriter that records every frame instead of sending it.
// Write frames are recorded as their JSON encoding, RawWrite frames as is.
type Writer struct {
	mu     sync.Mutex
	frames []string
	// Err, if set, is returned by every write (nothing is recorded).
	Err error
}

var _ wss.ConnWriter = (*Writer)(nil)

func (w *Writer) Write(_ context.Context, message any) error {
	if w.Err != nil {
		return w.Err
	}
	b, err := json.Marshal(message)
	if err != nil {
		return err
	}
	w.record(string(b))
	return nil
}

func (w *Writer) RawWrite(_ context.Context, message string) error {
	if w.Err != nil {
		return w.Err
	}
	w.record(message)
	return nil
}

func (w *Writer) record(frame string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.frames = append(w.frames, frame)
}

// Frames returns the recorded frames in order.
func (w *Writer) Frames() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.frames...)
}
