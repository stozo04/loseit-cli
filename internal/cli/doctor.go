package cli

import (
	"github.com/spf13/cobra"

	"github.com/stozo04/loseit-cli/internal/export"
	"github.com/stozo04/loseit-cli/internal/version"
)

// doctorReport is the doctor JSON shape. Key order frozen: tokenPresent,
// exportURL, tokenPath, configPath, version. There is no network check — Lose It
// has no OAuth to validate, and tokenPresent is a local file/env presence test
// that never reveals the cookie value.
type doctorReport struct {
	TokenPresent bool   `json:"tokenPresent"`
	ExportURL    string `json:"exportURL"`
	TokenPath    string `json:"tokenPath"`
	ConfigPath   string `json:"configPath"`
	Version      string `json:"version"`
}

// newDoctorCmd implements `doctor`: print config + whether a token is present as
// indent-2 JSON. No network. Lacking a token is not a failure — the --zip path
// needs none — so doctor always exits 0; it just hints when a token is missing.
func newDoctorCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Report config + whether a Lose It token is present (no network)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := app.resolveConfig()
			if err != nil {
				return err
			}

			present := export.HasToken(cfg)
			report := doctorReport{
				TokenPresent: present,
				ExportURL:    cfg.ExportURL,
				TokenPath:    cfg.TokenPath,
				ConfigPath:   cfg.ConfigPath,
				Version:      version.Info().Version,
			}
			if err := writeJSON(cmd.OutOrStdout(), report); err != nil {
				return err
			}

			if !present {
				fprintf(cmd.ErrOrStderr(),
					"\nNo Lose It token found. Pass --zip <export.zip> with a fresh download, "+
						"or save the liauth cookie to %s.\n", cfg.TokenPath)
			}
			return nil
		},
	}
}
