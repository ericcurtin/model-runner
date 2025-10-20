package progress

import (
	"fmt"
	"io"
	"sync"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// MultiLayerTracker tracks progress across multiple layers and aggregates them
type MultiLayerTracker struct {
	out         io.Writer
	format      progressF
	imageSize   uint64
	layers      map[string]*layerProgress
	mu          sync.Mutex
	done        chan struct{}
	err         error
	lastUpdate  time.Time
	lastTotal   uint64
	updateCount int
}

type layerProgress struct {
	id      string
	size    uint64
	current uint64
	updates chan v1.Update
}

// NewMultiLayerTracker creates a new multi-layer progress tracker
func NewMultiLayerTracker(w io.Writer, msgF progressF, imageSize int64) *MultiLayerTracker {
	return &MultiLayerTracker{
		out:       w,
		format:    msgF,
		imageSize: safeUint64(imageSize),
		layers:    make(map[string]*layerProgress),
		done:      make(chan struct{}),
	}
}

// AddLayer registers a new layer to track
func (t *MultiLayerTracker) AddLayer(layer v1.Layer) (chan<- v1.Update, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	id, err := layer.DiffID()
	if err != nil {
		return nil, fmt.Errorf("getting layer diff ID: %w", err)
	}

	size, err := layer.Size()
	if err != nil {
		return nil, fmt.Errorf("getting layer size: %w", err)
	}

	layerID := id.String()
	updates := make(chan v1.Update, 1)

	lp := &layerProgress{
		id:      layerID,
		size:    safeUint64(size),
		current: 0,
		updates: updates,
	}

	t.layers[layerID] = lp

	// Start goroutine to track this layer's progress
	go t.trackLayer(lp)

	return updates, nil
}

// trackLayer monitors a single layer's progress updates
func (t *MultiLayerTracker) trackLayer(lp *layerProgress) {
	for update := range lp.updates {
		t.mu.Lock()
		lp.current = safeUint64(update.Complete)
		t.mu.Unlock()

		// Trigger aggregated progress update
		t.reportProgress()
	}
}

// reportProgress calculates and reports the aggregated progress
func (t *MultiLayerTracker) reportProgress() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.out == nil || t.err != nil {
		return
	}

	now := time.Now()
	
	// Calculate total progress across all layers
	totalProgress := uint64(0)
	var currentLayerID string
	var currentLayerSize uint64
	var currentLayerProgress uint64

	for id, lp := range t.layers {
		totalProgress += lp.current
		// Use the most recently updated layer as the "current" layer for reporting
		if lp.current < lp.size {
			currentLayerID = id
			currentLayerSize = lp.size
			currentLayerProgress = lp.current
		}
	}

	// If no incomplete layers, use the last layer
	if currentLayerID == "" && len(t.layers) > 0 {
		for _, lp := range t.layers {
			currentLayerID = lp.id
			currentLayerSize = lp.size
			currentLayerProgress = lp.current
			break
		}
	}

	incrementalBytes := totalProgress - t.lastTotal

	// Only update if enough time has passed or enough bytes downloaded or finished
	if now.Sub(t.lastUpdate) >= UpdateInterval ||
		incrementalBytes >= MinBytesForUpdate ||
		totalProgress == t.imageSize {
		
		// Create a fake update for the format function to display total progress
		update := v1.Update{
			Complete: int64(totalProgress),
			Total:    int64(t.imageSize),
		}

		// Write progress using the current layer being downloaded (for per-layer tracking)
		// but use the total progress in the message
		if err := WriteProgress(t.out, t.format(update), t.imageSize, currentLayerSize, currentLayerProgress, currentLayerID); err != nil {
			t.err = err
		}
		t.lastUpdate = now
		t.lastTotal = totalProgress
		t.updateCount++
	}
}

// Wait waits for all layers to complete and returns any error encountered
func (t *MultiLayerTracker) Wait() error {
	// Wait a bit to ensure all pending updates are processed
	time.Sleep(50 * time.Millisecond)
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.err
}
