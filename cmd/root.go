package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/jmsperu/confedit/internal/parser"
	"github.com/spf13/cobra"
)

var appVersion string

func SetVersion(v string) { appVersion = v }

var rootCmd = &cobra.Command{
	Use:   "confedit <file>",
	Short: "Config file viewer, editor, and validator",
	Long: `confedit - Configuration File Editor

Parse, view, edit, and validate configuration files.
Supports YAML, JSON, INI, .env, SSH config, nginx, sysctl, /etc/hosts, and more.

Examples:
  confedit /etc/nginx/nginx.conf           # view parsed config
  confedit ~/.ssh/config                   # view SSH config
  confedit .env                            # view .env file
  confedit get .env DATABASE_URL           # get specific key
  confedit set .env DATABASE_URL "new_val" # set a value
  confedit validate config.yml             # validate syntax
  confedit diff config.yml config.prod.yml # compare configs
  confedit search .env "PASSWORD"          # search for keys`,
	Version: appVersion,
	Args:    cobra.ExactArgs(1),
	RunE:    runView,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringP("output", "o", "table", "Output format: table, json, flat")
	rootCmd.Flags().StringP("section", "s", "", "Filter by section")
	rootCmd.Flags().StringP("filter", "f", "", "Filter keys by substring")

	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(typeCmd)
}

func runView(cmd *cobra.Command, args []string) error {
	filename := args[0]
	outputFmt, _ := cmd.Flags().GetString("output")
	section, _ := cmd.Flags().GetString("section")
	filter, _ := cmd.Flags().GetString("filter")

	kvs, fileType, err := parser.ParseFile(filename)
	if err != nil {
		return err
	}

	// Apply filters
	if section != "" {
		var filtered []parser.KeyValue
		for _, kv := range kvs {
			if strings.EqualFold(kv.Section, section) || strings.Contains(strings.ToLower(kv.Section), strings.ToLower(section)) {
				filtered = append(filtered, kv)
			}
		}
		kvs = filtered
	}
	if filter != "" {
		var filtered []parser.KeyValue
		filterLower := strings.ToLower(filter)
		for _, kv := range kvs {
			if strings.Contains(strings.ToLower(kv.Key), filterLower) ||
				strings.Contains(strings.ToLower(kv.Value), filterLower) {
				filtered = append(filtered, kv)
			}
		}
		kvs = filtered
	}

	fmt.Printf("File: %s (type: %s)\n\n", filename, fileType)

	switch outputFmt {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(kvs)
	case "flat":
		for _, kv := range kvs {
			fmt.Printf("%s=%s\n", kv.Key, kv.Value)
		}
		return nil
	default:
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if hasSection(kvs) {
			fmt.Fprintf(w, "SECTION\tKEY\tVALUE\tLINE\n")
			fmt.Fprintf(w, "-------\t---\t-----\t----\n")
			for _, kv := range kvs {
				value := kv.Value
				if len(value) > 60 {
					value = value[:60] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", kv.Section, kv.Key, value, kv.Line)
			}
		} else {
			fmt.Fprintf(w, "KEY\tVALUE\tLINE\n")
			fmt.Fprintf(w, "---\t-----\t----\n")
			for _, kv := range kvs {
				value := kv.Value
				if len(value) > 70 {
					value = value[:70] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%d\n", kv.Key, value, kv.Line)
			}
		}
		w.Flush()
		fmt.Printf("\n%d entries\n", len(kvs))
		return nil
	}
}

func hasSection(kvs []parser.KeyValue) bool {
	for _, kv := range kvs {
		if kv.Section != "" {
			return true
		}
	}
	return false
}

var getCmd = &cobra.Command{
	Use:   "get <file> <key>",
	Short: "Get a specific config value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kvs, _, err := parser.ParseFile(args[0])
		if err != nil {
			return err
		}

		keyLower := strings.ToLower(args[1])
		for _, kv := range kvs {
			if strings.ToLower(kv.Key) == keyLower {
				fmt.Println(kv.Value)
				return nil
			}
		}
		return fmt.Errorf("key %q not found", args[1])
	},
}

var setCmd = &cobra.Command{
	Use:   "set <file> <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := parser.SetValue(args[0], args[1], args[2]); err != nil {
			return err
		}
		fmt.Printf("Set %s = %s\n", args[1], args[2])
		return nil
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate config file syntax",
	Aliases: []string{"check"},
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		results := parser.Validate(args[0])
		for _, r := range results {
			fmt.Println(r)
		}
		return nil
	},
}

var diffCmd = &cobra.Command{
	Use:   "diff <file1> <file2>",
	Short: "Compare two config files",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kvs1, _, err := parser.ParseFile(args[0])
		if err != nil {
			return fmt.Errorf("file 1: %w", err)
		}
		kvs2, _, err := parser.ParseFile(args[1])
		if err != nil {
			return fmt.Errorf("file 2: %w", err)
		}

		map1 := make(map[string]string)
		map2 := make(map[string]string)
		for _, kv := range kvs1 {
			map1[kv.Key] = kv.Value
		}
		for _, kv := range kvs2 {
			map2[kv.Key] = kv.Value
		}

		fmt.Printf("Comparing %s vs %s\n\n", args[0], args[1])

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "STATUS\tKEY\tFILE1\tFILE2\n")
		fmt.Fprintf(w, "------\t---\t-----\t-----\n")

		allKeys := make(map[string]bool)
		for k := range map1 {
			allKeys[k] = true
		}
		for k := range map2 {
			allKeys[k] = true
		}

		changes := 0
		for key := range allKeys {
			v1, in1 := map1[key]
			v2, in2 := map2[key]

			if in1 && !in2 {
				fmt.Fprintf(w, "REMOVED\t%s\t%s\t-\n", key, truncate(v1, 30))
				changes++
			} else if !in1 && in2 {
				fmt.Fprintf(w, "ADDED\t%s\t-\t%s\n", key, truncate(v2, 30))
				changes++
			} else if v1 != v2 {
				fmt.Fprintf(w, "CHANGED\t%s\t%s\t%s\n", key, truncate(v1, 30), truncate(v2, 30))
				changes++
			}
		}
		w.Flush()

		if changes == 0 {
			fmt.Println("Files are identical.")
		} else {
			fmt.Printf("\n%d differences\n", changes)
		}
		return nil
	},
}

var searchCmd = &cobra.Command{
	Use:   "search <file> <pattern>",
	Short: "Search for keys or values matching a pattern",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kvs, _, err := parser.ParseFile(args[0])
		if err != nil {
			return err
		}

		pattern := strings.ToLower(args[1])
		found := 0

		for _, kv := range kvs {
			if strings.Contains(strings.ToLower(kv.Key), pattern) ||
				strings.Contains(strings.ToLower(kv.Value), pattern) {
				fmt.Printf("  %s = %s (line %d)\n", kv.Key, kv.Value, kv.Line)
				found++
			}
		}

		if found == 0 {
			fmt.Println("No matches found.")
		} else {
			fmt.Printf("\n%d matches\n", found)
		}
		return nil
	},
}

var typeCmd = &cobra.Command{
	Use:   "type <file>",
	Short: "Detect config file type",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ft := parser.DetectType(args[0])
		fmt.Printf("%s: %s\n", args[0], ft)
		return nil
	},
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
