package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/samsiva-dev/error-cemetery/internal/ai"
	"github.com/samsiva-dev/error-cemetery/internal/config"
	"github.com/samsiva-dev/error-cemetery/internal/db"
	"github.com/samsiva-dev/error-cemetery/internal/match"
	"github.com/samsiva-dev/error-cemetery/internal/tui"
	"github.com/spf13/cobra"
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
		initCmd(),
		buryCmd(),
		unburyCmd(),
		digCmd(),
		visitCmd(),
		statsCmd(),
		configCmd(),
		exportCmd(),
	)
	return root
}

// ── init ──────────────────────────────────────────────────────────────────────

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialise config and database for first use",
		Long: `Creates the config file and database if they do not exist.
Safe to run multiple times — existing files are never overwritten.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// Write config only if it doesn't already exist.
			cfgPath := config.DefaultPath()
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				if err := config.Write(cfg); err != nil {
					return fmt.Errorf("write config: %w", err)
				}
				fmt.Printf("  Created config:   %s\n", cfgPath)
			} else {
				fmt.Printf("  Config exists:    %s\n", cfgPath)
			}

			// Open DB — db.Open creates the directory and applies the schema.
			store, err := openStore(cfg)
			if err != nil {
				return fmt.Errorf("init database: %w", err)
			}
			store.Close()
			fmt.Printf("  Database ready:   %s\n", cfg.Cemetery.DBPath)

			fmt.Println()
			fmt.Println("⚰  Cemetery ready. Run `cemetery bury` to bury your first error.")
			return nil
		},
	}
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

// ── unbury ───────────────────────────────────────────────────────────────────

func unburyCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "unbury <id>",
		Short: "Permanently delete a buried error by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil || id <= 0 {
				return fmt.Errorf("invalid id %q — must be a positive integer", args[0])
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			store, err := openStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			burial, err := store.GetByID(id)
			if err != nil {
				return fmt.Errorf("no entry with id %d", id)
			}

			if !force {
				fmt.Printf("Unbury id %d: %s\n", burial.ID, firstLine(burial.ErrorText))
				fmt.Print("Are you sure? [y/N] ")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			if err := store.Delete(id); err != nil {
				return fmt.Errorf("delete: %w", err)
			}
			fmt.Printf("⚰  Entry %d removed.\n", id)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
	return cmd
}

// ── helpers ───────────────────────────────────────────────────────────────────

func openStore(cfg *config.Config) (*db.Store, error) {
	return db.Open(cfg.Cemetery.DBPath)
}

// ── export ────────────────────────────────────────────────────────────────────

func exportCmd() *cobra.Command {
	var outPath string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export all buried errors to a Markdown file",
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
				fmt.Println("The graveyard is empty — nothing to export.")
				return nil
			}

			f, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}
			defer f.Close()

			fmt.Fprintf(f, "# ⚰ Error Cemetery Export\n\n")
			fmt.Fprintf(f, "_%d entries — exported %s_\n\n", len(burials), burials[0].BuriedAt.Format("2006-01-02"))
			fmt.Fprintf(f, "---\n\n")

			for _, b := range burials {
				fmt.Fprintf(f, "## %d. %s\n\n", b.ID, firstLine(b.ErrorText))
				fmt.Fprintf(f, "**Buried:** %s", b.BuriedAt.Format("2006-01-02 15:04"))
				if b.Tags != "" {
					fmt.Fprintf(f, " &nbsp;·&nbsp; **Tags:** %s", b.Tags)
				}
				if b.TimesDug > 0 {
					fmt.Fprintf(f, " &nbsp;·&nbsp; **Dug:** %d×", b.TimesDug)
				}
				fmt.Fprintf(f, "\n\n")

				fmt.Fprintf(f, "### Error\n\n```\n%s\n```\n\n", b.ErrorText)
				fmt.Fprintf(f, "### Fix\n\n%s\n\n", b.FixText)

				if b.Context != "" {
					fmt.Fprintf(f, "### Context\n\n%s\n\n", b.Context)
				}

				fmt.Fprintf(f, "---\n\n")
			}

			fmt.Printf("⚰  Exported %d entries → %s\n", len(burials), outPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outPath, "out", "o", "cemetery-export.md", "output file path")
	return cmd
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i != -1 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
