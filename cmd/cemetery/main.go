package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
	"github.com/samsiva-dev/error-cemetery/internal/ai"
	"github.com/samsiva-dev/error-cemetery/internal/config"
	"github.com/samsiva-dev/error-cemetery/internal/db"
	"github.com/samsiva-dev/error-cemetery/internal/match"
	"github.com/samsiva-dev/error-cemetery/internal/tui"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "cemetery",
		Short: "⚰  Error Cemetery — bury errors, dig them up later",
		Long: `Error Cemetery lets you bury errors when you fix them
and dig them up when history repeats itself.`,
	}

	root.AddCommand(
		buryCmd(),
		digCmd(),
		visitCmd(),
		statsCmd(),
		configCmd(),
	)
	return root
}

// ── bury ──────────────────────────────────────────────────────────────────────

func buryCmd() *cobra.Command {
	var fromClip bool

	cmd := &cobra.Command{
		Use:   "bury",
		Short: "Bury an error and its fix",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			store, err := openStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			prefill := ""
			if fromClip {
				prefill, err = clipboard.ReadAll()
				if err != nil {
					return fmt.Errorf("clipboard read: %w", err)
				}
			}

			tags, _ := store.AllTags()
			result, err := tui.RunBury(prefill, tags)
			if err != nil {
				return err
			}
			if !result.Submitted {
				fmt.Println("Burial cancelled.")
				return nil
			}

			if strings.TrimSpace(result.Input.ErrorText) == "" {
				return fmt.Errorf("error text cannot be empty")
			}
			if strings.TrimSpace(result.Input.FixText) == "" {
				return fmt.Errorf("fix text cannot be empty")
			}

			burial, err := store.Bury(result.Input)
			if err != nil {
				return fmt.Errorf("bury: %w", err)
			}
			fmt.Printf("\n⚰  Buried. (id %d)\n", burial.ID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&fromClip, "clip", false, "pre-fill error from clipboard")
	return cmd
}

// ── dig ───────────────────────────────────────────────────────────────────────

func digCmd() *cobra.Command {
	var fromClip bool
	var smart bool

	cmd := &cobra.Command{
		Use:   "dig [query]",
		Short: "Search for a matching buried error",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			store, err := openStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			var query string
			if fromClip {
				query, err = clipboard.ReadAll()
				if err != nil {
					return fmt.Errorf("clipboard read: %w", err)
				}
			} else if len(args) > 0 {
				query = args[0]
			} else {
				return fmt.Errorf("provide a query or use --clip")
			}

			useSmart := smart || cfg.Cemetery.SmartMode
			var aiClient *ai.Client
			if useSmart && cfg.Claude.APIKey != "" {
				aiClient = ai.NewClient(cfg.Claude.APIKey, cfg.Claude.Model)
			}

			results, err := match.Rank(query, store, aiClient, useSmart)
			if err != nil {
				return fmt.Errorf("rank: %w", err)
			}

			if len(results) == 0 {
				fmt.Println("No matching graves found. Try `cemetery bury` to add one.")
				return nil
			}

			return tui.RunDig(results)
		},
	}

	cmd.Flags().BoolVar(&fromClip, "clip", false, "use clipboard content as query")
	cmd.Flags().BoolVar(&smart, "smart", false, "use Claude semantic search (Pass 3)")
	return cmd
}

// ── visit ─────────────────────────────────────────────────────────────────────

func visitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "visit",
		Short: "Browse the full graveyard",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			store, err := openStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			burials, err := store.GetAll()
			if err != nil {
				return err
			}
			if len(burials) == 0 {
				fmt.Println("The graveyard is empty. Use `cemetery bury` to add your first error.")
				return nil
			}
			return tui.RunVisit(burials)
		},
	}
}

// ── stats ─────────────────────────────────────────────────────────────────────

func statsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show cemetery statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			store, err := openStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			total, topTags, err := store.Stats()
			if err != nil {
				return err
			}

			fmt.Printf("\n⚰  Cemetery Stats\n\n")
			fmt.Printf("  Total buried: %d\n", total)
			if len(topTags) > 0 {
				fmt.Printf("\n  Top tags:\n")
				for i, tc := range topTags {
					if i >= 10 {
						break
					}
					fmt.Printf("    %-20s %d×\n", tc.Tag, tc.Count)
				}
			}
			fmt.Println()
			return nil
		},
	}
}

// ── config ────────────────────────────────────────────────────────────────────

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Open or show config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := config.DefaultPath()
			// ensure file exists
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := config.Write(cfg); err != nil {
				return err
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "nano"
			}

			fmt.Printf("Config: %s\n", path)
			c := exec.Command(editor, path)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
	return cmd
}

// ── helpers ───────────────────────────────────────────────────────────────────

func openStore(cfg *config.Config) (*db.Store, error) {
	return db.Open(cfg.Cemetery.DBPath)
}
