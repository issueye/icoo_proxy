package ai_llm_proxy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/issueye/icoo_proxy/common/constants"
)

func TestConvertStreamSameProtocolStopsWhenContextCanceled(t *testing.T) {
	reader := newBlockingStreamReader(nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		_, err := NewProtocolConverter().ConvertStream(StreamInput{
			Context:    ctx,
			Downstream: constants.ProtocolOpenAIChat,
			Upstream:   constants.ProtocolOpenAIChat,
			Reader:     reader,
			Writer:     io.Discard,
		})
		done <- err
	}()

	waitForRead(t, reader.started)
	cancel()
	assertCanceledPromptly(t, done)
	waitForRead(t, reader.stopped)
}

func TestConvertStreamCrossProtocolStopsSSEScanWhenContextCanceled(t *testing.T) {
	reader := newBlockingStreamReader([]byte("event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\n"))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		_, err := NewProtocolConverter().ConvertStream(StreamInput{
			Context:    ctx,
			Downstream: constants.ProtocolOpenAIChat,
			Upstream:   constants.ProtocolOpenAIResponses,
			Model:      "test-model",
			Reader:     reader,
			Writer:     &bytes.Buffer{},
		})
		done <- err
	}()

	waitForRead(t, reader.started)
	cancel()
	assertCanceledPromptly(t, done)
	waitForRead(t, reader.stopped)
}

type blockingStreamReader struct {
	data    *bytes.Reader
	started chan struct{}
	stopped chan struct{}
	closed  chan struct{}
	once    sync.Once
}

func newBlockingStreamReader(data []byte) *blockingStreamReader {
	return &blockingStreamReader{
		data:    bytes.NewReader(data),
		started: make(chan struct{}),
		stopped: make(chan struct{}),
		closed:  make(chan struct{}),
	}
}

func (r *blockingStreamReader) Read(p []byte) (int, error) {
	if r.data.Len() > 0 {
		return r.data.Read(p)
	}
	r.once.Do(func() { close(r.started) })
	<-r.closed
	close(r.stopped)
	return 0, io.ErrClosedPipe
}

func (r *blockingStreamReader) Close() error {
	select {
	case <-r.closed:
	default:
		close(r.closed)
	}
	return nil
}

func waitForRead(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("stream reader did not stop promptly")
	}
}

func assertCanceledPromptly(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ConvertStream() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ConvertStream() did not return promptly after cancellation")
	}
}
