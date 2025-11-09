package desktop

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-units"
	"github.com/mattn/go-isatty"
)

// DisplayProgress displays progress messages from a model pull/push operation
// using Docker-style multi-line progress bars
func DisplayProgress(body io.Reader, printer StatusPrinter) (string, error) {
	fd, isTerminal := printer.GetFdInfo()

	// If not a terminal, fall back to simple line-by-line output
	if !isTerminal {
		return displayProgressSimple(body, printer)
	}

	// Use a pipe to convert our progress messages to Docker's JSONMessage format
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)

	// Start the display goroutine
	go func() {
		err := jsonmessage.DisplayJSONMessagesStream(pr, &writerAdapter{printer}, fd, isTerminal, nil)
		if err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	// Convert progress messages to JSONMessage format
	scanner := bufio.NewScanner(body)
	layerStatus := make(map[string]string) // Track status of each layer
	var finalMessage string

	for scanner.Scan() {
		progressLine := scanner.Text()
		if progressLine == "" {
			continue
		}

		var progressMsg ProgressMessage
		if err := json.Unmarshal([]byte(html.UnescapeString(progressLine)), &progressMsg); err != nil {
			// If we can't parse, just skip
			continue
		}

		switch progressMsg.Type {
		case "progress":
			if err := writeDockerProgress(pw, &progressMsg, layerStatus); err != nil {
				pw.Close()
				return "", err
			}

		case "success":
			finalMessage = progressMsg.Message
			// Write final success message
			if err := writeDockerStatus(pw, "", "success", progressMsg.Message); err != nil {
				pw.Close()
				return "", err
			}

		case "error":
			pw.Close()
			return "", fmt.Errorf("%s", progressMsg.Message)
		}
	}

	if err := scanner.Err(); err != nil {
		pw.Close()
		return "", err
	}

	pw.Close()

	// Wait for display to finish
	if err := <-errCh; err != nil && err != io.EOF {
		return finalMessage, err
	}

	return finalMessage, nil
}

// displayProgressSimple displays progress messages in simple line-by-line format
func displayProgressSimple(body io.Reader, printer StatusPrinter) (string, error) {
	scanner := bufio.NewScanner(body)
	current := uint64(0)
	layerProgress := make(map[string]uint64)
	var finalMessage string

	for scanner.Scan() {
		progressLine := scanner.Text()
		if progressLine == "" {
			continue
		}

		var progressMsg ProgressMessage
		if err := json.Unmarshal([]byte(html.UnescapeString(progressLine)), &progressMsg); err != nil {
			continue
		}

		switch progressMsg.Type {
		case "progress":
			layerID := progressMsg.Layer.ID
			layerProgress[layerID] = progressMsg.Layer.Current

			// Sum all layer progress
			current = uint64(0)
			for _, layerCurrent := range layerProgress {
				current += layerCurrent
			}

			printer.Println(fmt.Sprintf("Downloaded %s of %s",
				units.CustomSize("%.2f%s", float64(current), 1000.0, []string{"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}),
				units.CustomSize("%.2f%s", float64(progressMsg.Total), 1000.0, []string{"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"})))

		case "success":
			finalMessage = progressMsg.Message

		case "error":
			return "", fmt.Errorf("%s", progressMsg.Message)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return finalMessage, nil
}

// writeDockerProgress writes a progress update in Docker's JSONMessage format
func writeDockerProgress(w io.Writer, msg *ProgressMessage, layerStatus map[string]string) error {
	layerID := msg.Layer.ID
	if layerID == "" {
		return nil
	}

	// Determine status based on progress
	var status string
	var progressDetail *jsonmessage.JSONProgress

	if msg.Layer.Current == 0 {
		status = "Waiting"
	} else if msg.Layer.Current < msg.Layer.Size {
		status = "Downloading"
		progressDetail = &jsonmessage.JSONProgress{
			Current: int64(msg.Layer.Current),
			Total:   int64(msg.Layer.Size),
		}
	} else if msg.Layer.Current >= msg.Layer.Size && msg.Layer.Size > 0 {
		// Check if we've already marked this as complete
		if layerStatus[layerID] != "Download complete" {
			status = "Download complete"
			layerStatus[layerID] = status
		} else {
			// Already shown complete, skip
			return nil
		}
	}

	if status == "" {
		return nil
	}

	// Shorten layer ID for display (similar to Docker)
	displayID := strings.TrimPrefix(layerID, "sha256:")
	if len(displayID) > 12 {
		displayID = displayID[:12]
	}

	dockerMsg := jsonmessage.JSONMessage{
		ID:       displayID,
		Status:   status,
		Progress: progressDetail,
	}

	data, err := json.Marshal(dockerMsg)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

// writeDockerStatus writes a status message in Docker's JSONMessage format
func writeDockerStatus(w io.Writer, id, status, message string) error {
	msg := jsonmessage.JSONMessage{
		ID:     id,
		Status: status,
	}

	if message != "" {
		msg.Status = message
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

// StatusPrinter is an interface for printing status messages
type StatusPrinter interface {
	GetFdInfo() (uintptr, bool)
	Println(...interface{}) (int, error)
}

// writerAdapter adapts StatusPrinter to io.Writer for jsonmessage
type writerAdapter struct {
	printer StatusPrinter
}

func (w *writerAdapter) Write(p []byte) (n int, err error) {
	return w.printer.Println(string(p))
}

// cobraPrinter adapts a cobra.Command to the StatusPrinter interface
type cobraPrinter struct {
	outFile *os.File
}

func (p *cobraPrinter) GetFdInfo() (uintptr, bool) {
	if p.outFile == nil {
		return 0, false
	}
	return p.outFile.Fd(), isatty.IsTerminal(p.outFile.Fd())
}

func (p *cobraPrinter) Println(a ...interface{}) (int, error) {
	s := fmt.Sprint(a...)
	n, err := p.outFile.Write([]byte(s))
	return n, err
}

// NewCobraPrinter creates a StatusPrinter that writes to stdout
func NewCobraPrinter() StatusPrinter {
	return &cobraPrinter{
		outFile: os.Stdout,
	}
}

// simplePrinter is a simple StatusPrinter that just writes to a function
type simplePrinter struct {
	printFunc func(string)
}

func (p *simplePrinter) GetFdInfo() (uintptr, bool) {
	return 0, false
}

func (p *simplePrinter) Println(a ...interface{}) (int, error) {
	s := fmt.Sprint(a...)
	p.printFunc(s)
	return len(s), nil
}

// NewSimplePrinter creates a StatusPrinter from a simple print function
func NewSimplePrinter(printFunc func(string)) StatusPrinter {
	return &simplePrinter{
		printFunc: printFunc,
	}
}
