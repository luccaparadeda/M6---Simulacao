package logger

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Entry struct {
	RNGCount    int            `json:"rng_count"`
	Time        float64        `json:"time"`
	Event       string         `json:"event"`
	QueueID     string         `json:"queue"`
	Detail      string         `json:"detail"`
	Populations map[string]int `json:"populations"`
	Losses      map[string]int `json:"losses"`
}

type Logger interface {
	Log(e Entry)
	Close()
}

// CSVLogger writes one row per event to a CSV file.
type CSVLogger struct {
	file     *os.File
	writer   *csv.Writer
	queueIDs []string
}

func NewCSV(path string, queueIDs []string) (*CSVLogger, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	w := csv.NewWriter(f)

	header := []string{"rng_count", "time", "event", "queue", "detail"}
	for _, id := range queueIDs {
		header = append(header, id+"_pop", id+"_losses")
	}
	if err := w.Write(header); err != nil {
		f.Close()
		return nil, err
	}

	return &CSVLogger{file: f, writer: w, queueIDs: queueIDs}, nil
}

func (l *CSVLogger) Log(e Entry) {
	row := []string{
		fmt.Sprintf("%d", e.RNGCount),
		fmt.Sprintf("%.4f", e.Time),
		e.Event,
		e.QueueID,
		e.Detail,
	}
	for _, id := range l.queueIDs {
		row = append(row,
			fmt.Sprintf("%d", e.Populations[id]),
			fmt.Sprintf("%d", e.Losses[id]),
		)
	}
	l.writer.Write(row)
}

func (l *CSVLogger) Close() {
	l.writer.Flush()
	l.file.Close()
}

// JSONLogger writes one JSON object per line (JSONL) to a file.
type JSONLogger struct {
	file    *os.File
	encoder *json.Encoder
}

func NewJSON(path string) (*JSONLogger, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &JSONLogger{file: f, encoder: json.NewEncoder(f)}, nil
}

func (l *JSONLogger) Log(e Entry) {
	l.encoder.Encode(e)
}

func (l *JSONLogger) Close() {
	l.file.Close()
}

func FormatRoute(from, to string) string {
	if to == "" {
		return from + " -> EXIT"
	}
	return from + " -> " + to
}

func FormatState(pops map[string]int, ids []string) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%s=%d", id, pops[id]))
	}
	return strings.Join(parts, " ")
}
