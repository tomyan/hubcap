package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// TakeHeapSnapshot captures a V8 heap snapshot and writes it to a file.
func (c *Client) TakeHeapSnapshot(ctx context.Context, targetID string) ([]byte, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable HeapProfiler
	_, err = c.CallSession(ctx, sessionID, "HeapProfiler.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling HeapProfiler: %w", err)
	}

	// Subscribe to chunk events before taking snapshot
	chunkCh := c.subscribeEvent(sessionID, "HeapProfiler.addHeapSnapshotChunk")

	// Collect chunks in a goroutine
	var chunks []string
	var chunksMu sync.Mutex
	chunksDone := make(chan struct{})

	go func() {
		defer close(chunksDone)
		for params := range chunkCh {
			var chunk struct {
				Chunk string `json:"chunk"`
			}
			if err := json.Unmarshal(params, &chunk); err == nil {
				chunksMu.Lock()
				chunks = append(chunks, chunk.Chunk)
				chunksMu.Unlock()
			}
		}
	}()

	// Take the snapshot (blocks until complete)
	_, err = c.CallSession(ctx, sessionID, "HeapProfiler.takeHeapSnapshot", map[string]interface{}{
		"reportProgress": false,
	})

	// Unsubscribe closes the channel, which signals the goroutine to finish
	c.unsubscribeEvent(sessionID, "HeapProfiler.addHeapSnapshotChunk", chunkCh)
	<-chunksDone

	// Disable HeapProfiler
	c.CallSession(ctx, sessionID, "HeapProfiler.disable", nil)

	if err != nil {
		return nil, fmt.Errorf("taking heap snapshot: %w", err)
	}

	// Assemble chunks
	chunksMu.Lock()
	defer chunksMu.Unlock()

	var buf strings.Builder
	for _, chunk := range chunks {
		buf.WriteString(chunk)
	}

	return []byte(buf.String()), nil
}

// CaptureTrace captures a Chrome performance trace for the given duration.
func (c *Client) CaptureTrace(ctx context.Context, targetID string, duration time.Duration) ([]byte, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Subscribe to tracing events
	dataCh := c.subscribeEvent(sessionID, "Tracing.dataCollected")
	defer c.unsubscribeEvent(sessionID, "Tracing.dataCollected", dataCh)

	completeCh := c.subscribeEvent(sessionID, "Tracing.tracingComplete")
	defer c.unsubscribeEvent(sessionID, "Tracing.tracingComplete", completeCh)

	// Collect trace data chunks
	var traceEvents []json.RawMessage
	var traceEventsMu sync.Mutex
	collectDone := make(chan struct{})

	go func() {
		for {
			select {
			case params, ok := <-dataCh:
				if !ok {
					return
				}
				var data struct {
					Value []json.RawMessage `json:"value"`
				}
				if err := json.Unmarshal(params, &data); err == nil {
					traceEventsMu.Lock()
					traceEvents = append(traceEvents, data.Value...)
					traceEventsMu.Unlock()
				}
			case <-collectDone:
				return
			}
		}
	}()

	// Start tracing
	_, err = c.CallSession(ctx, sessionID, "Tracing.start", map[string]interface{}{
		"categories": "-*,devtools.timeline,v8.execute,disabled-by-default-devtools.timeline",
	})
	if err != nil {
		close(collectDone)
		return nil, fmt.Errorf("starting trace: %w", err)
	}

	// Wait for the specified duration
	time.Sleep(duration)

	// End tracing
	_, err = c.CallSession(ctx, sessionID, "Tracing.end", nil)
	if err != nil {
		close(collectDone)
		return nil, fmt.Errorf("ending trace: %w", err)
	}

	// Wait for tracing complete event
	select {
	case <-completeCh:
	case <-time.After(10 * time.Second):
	case <-ctx.Done():
		close(collectDone)
		return nil, ctx.Err()
	}

	close(collectDone)
	time.Sleep(100 * time.Millisecond)

	// Build JSON array from collected events
	traceEventsMu.Lock()
	defer traceEventsMu.Unlock()

	traceData, err := json.Marshal(traceEvents)
	if err != nil {
		return nil, fmt.Errorf("marshaling trace data: %w", err)
	}

	return traceData, nil
}
