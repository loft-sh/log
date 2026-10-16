package table

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/loft-sh/log"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/sirupsen/logrus"
)

func TestPrintTablePreservesLegacyStyle(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	previousNoColor := color.NoColor
	color.NoColor = false
	t.Cleanup(func() { color.NoColor = previousNoColor })
	if got := color.New(color.FgGreen).Sprint("NAME"); got == "NAME" {
		t.Fatal("test setup did not enable color")
	}

	var output bytes.Buffer
	logger := log.NewStreamLoggerWithFormat(&output, &output, logrus.InfoLevel, log.RawFormat)

	PrintTable(logger,
		[]string{"NAME", "NAMESPACE", "STATUS", "AGE"},
		[][]string{{"my-vcluster", "team-a", "Running", "3d"}, {"other", "team-b", "Paused", "17h"}},
	)

	got := output.String()
	if strings.ContainsAny(got, "┌┬┐├┼┤└┴┘│─") {
		t.Fatalf("expected ASCII table without outer borders, got %q", got)
	}
	if !strings.Contains(got, "-------------+-----------+---------+-----") {
		t.Fatalf("expected legacy ASCII header separator, got %q", got)
	}
	if !strings.Contains(got, "\x1b[32m") {
		t.Fatalf("expected green header, got %q", got)
	}
	if strings.Contains(got, "\x1b[36m") {
		t.Fatalf("expected uncolored rows, got %q", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "my-vcluster") && strings.Contains(line, "\x1b[") {
			t.Fatalf("expected uncolored rows, got %q", got)
		}
	}
}

func TestPrintTableOmitsColorWhenDisabled(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	previousNoColor := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = previousNoColor })

	var output bytes.Buffer
	logger := log.NewStreamLoggerWithFormat(&output, &output, logrus.InfoLevel, log.RawFormat)

	PrintTable(logger, []string{"NAME"}, [][]string{{"my-vcluster"}})

	got := output.String()
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("expected plain output when color is disabled, got %q", got)
	}
	if strings.ContainsAny(got, "┌┬┐├┼┤└┴┘│─") {
		t.Fatalf("expected ASCII table without outer borders, got %q", got)
	}
}

func TestPrintTableWithOptionsLogsRenderErrors(t *testing.T) {
	var output bytes.Buffer
	logger := log.NewStreamLoggerWithFormat(&output, &output, logrus.DebugLevel, log.RawFormat)

	PrintTableWithOptions(logger, []string{"NAME"}, [][]string{{"my-vcluster"}}, func(table *tablewriter.Table) {
		table.Options(tablewriter.WithRenderer(failingRenderer{Renderer: renderer.NewBlueprint()}))
	})

	if !strings.Contains(output.String(), "render table:") || !strings.Contains(output.String(), "render failed") {
		t.Fatalf("expected render error in debug output, got %q", output.String())
	}
}

type failingRenderer struct {
	tw.Renderer
}

func (failingRenderer) Start(io.Writer) error { return errors.New("render failed") }
