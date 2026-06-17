package cli

import (
	"github.com/spf13/cobra"
)

// configView is the stable shape emitted by `config show --json`. Key order
// frozen: token_path, export_url, config_path. There is no secret to redact —
// the token lives in a separate file, never in config.json.
type configView struct {
	TokenPath  string `json:"token_path"`
	ExportURL  string `json:"export_url"`
	ConfigPath string `json:"config_path"`
}

// newConfigCmd implements `config [show|path]`, a convenience for inspecting the
// resolved configuration without hand-reading config.json.
func newConfigCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect configuration (show/path)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newConfigShowCmd(app), newConfigPathCmd(app))
	return cmd
}

func newConfigShowCmd(app *App) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Print the resolved effective config",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := app.resolveConfig()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if asJSON {
				return writeJSON(out, configView{
					TokenPath:  cfg.TokenPath,
					ExportURL:  cfg.ExportURL,
					ConfigPath: cfg.ConfigPath,
				})
			}
			fprintf(out, "token_path:  %s\n", cfg.TokenPath)
			fprintf(out, "export_url:  %s\n", cfg.ExportURL)
			fprintf(out, "config_path: %s\n", cfg.ConfigPath)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newConfigPathCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config.json path in use",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := app.resolveConfig()
			if err != nil {
				return err
			}
			fprintln(cmd.OutOrStdout(), cfg.ConfigPath)
			return nil
		},
	}
}
