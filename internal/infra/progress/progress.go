package progress

import (
	"fmt"
	"io"
	"strings"
	"time"
)

const renderInterval = 200 * time.Millisecond

type Writer struct {
	out        io.Writer
	label      string
	total      int64
	written    int64
	startedAt  time.Time
	lastRender time.Time
	lastLine   int
}

func NewWriter(out io.Writer, label string, total int64) *Writer {
	if label == "" {
		label = "下载进度"
	}
	if total < 0 {
		total = 0
	}
	return &Writer{
		out:       out,
		label:     label,
		total:     total,
		startedAt: time.Now(),
	}
}

func (w *Writer) Write(p []byte) (int, error) {
	n := len(p)
	if n == 0 {
		return 0, nil
	}

	w.written += int64(n)
	now := time.Now()
	if w.lastRender.IsZero() || now.Sub(w.lastRender) >= renderInterval || (w.total > 0 && w.written >= w.total) {
		w.render(now)
	}
	return n, nil
}

func (w *Writer) Finish() {
	w.render(time.Now())
	fmt.Fprintln(w.out)
}

func (w *Writer) render(now time.Time) {
	line := w.progressLine(now)
	padding := ""
	if len(line) < w.lastLine {
		padding = strings.Repeat(" ", w.lastLine-len(line))
	}

	fmt.Fprintf(w.out, "\r%s%s", line, padding)
	w.lastLine = len(line)
	w.lastRender = now
}

func (w *Writer) progressLine(now time.Time) string {
	elapsed := now.Sub(w.startedAt).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}

	speed := int64(float64(w.written) / elapsed)
	if w.total <= 0 {
		return fmt.Sprintf("%s: %s (%s/s)", w.label, formatBytes(w.written), formatBytes(speed))
	}

	percent := float64(w.written) * 100 / float64(w.total)
	if percent > 100 {
		percent = 100
	}
	return fmt.Sprintf(
		"%s: %5.1f%% (%s/%s, %s/s)",
		w.label,
		percent,
		formatBytes(w.written),
		formatBytes(w.total),
		formatBytes(speed),
	)
}

func formatBytes(value int64) string {
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}

	units := []string{"KiB", "MiB", "GiB", "TiB"}
	size := float64(value)
	unit := "B"
	for _, current := range units {
		size /= 1024
		unit = current
		if size < 1024 {
			break
		}
	}
	return fmt.Sprintf("%.1f %s", size, unit)
}
