package commands

import (
	"bytes"

	"github.com/docker/go-units"
	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/docker/model-runner/cmd/cli/desktop"
	"github.com/spf13/cobra"
)

func newDFCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "df",
		Short: "Show Docker Model Runner disk usage",
		RunE: func(cmd *cobra.Command, args []string) error {
			df, err := desktopClient.DF()
			if err != nil {
				return handleClientError(err, "Failed to list running models")
			}
			cmd.Print(diskUsageTable(df))
			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	return c
}

func diskUsageTable(df desktop.DiskUsage) string {
	var buf bytes.Buffer
	table := newTable(&buf)
	table.Header([]string{"TYPE", "SIZE"})

	table.Append([]string{"Models", units.CustomSize("%.2f%s", float64(df.ModelsDiskUsage), 1000.0, []string{"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"})})
	if df.DefaultBackendDiskUsage != 0 {
		table.Append([]string{"Inference engine", units.CustomSize("%.2f%s", float64(df.DefaultBackendDiskUsage), 1000.0, []string{"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"})})
	}

	table.Render()
	return buf.String()
}
