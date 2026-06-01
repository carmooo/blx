package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/joao-carmo/blx/internal/repository/ipac"
	"github.com/joao-carmo/blx/internal/service"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "blx",
		Short: "BLX - Lisbon public libraries catalog CLI",
	}
	root.AddCommand(searchCmd())
	return root
}

func searchCmd() *cobra.Command {
	var (
		index    string
		sort     string
		branch   string
		lang     string
		page     int
		jsonOut  bool
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search the Lisbon public libraries catalog",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			query := args[0]

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			client := ipac.NewClient()
			repo := ipac.NewRepository(client)

			params := service.SearchParams{
				Query:    query,
				Index:    index,
				Sort:     sort,
				Branch:   branch,
				Language: lang,
				Page:     page,
			}

			result, err := repo.Search(ctx, params)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

			if jsonOut {
				return printJSON(result)
			}
			return printTable(result)
		},
	}

	cmd.Flags().StringVar(&index, "index", "keyword", "Search index (keyword, title, author, subject, collection, publisher, place)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (newest, oldest, title_az)")
	cmd.Flags().StringVar(&branch, "branch", "", "Branch code (e.g. CMLBC4)")
	cmd.Flags().StringVar(&lang, "lang", "", "Language code (e.g. por)")
	cmd.Flags().IntVar(&page, "page", 1, "Results page number")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output results as JSON")

	return cmd
}

func printTable(result *service.SearchResult) error {
	if len(result.Results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE")
	fmt.Fprintln(w, "--\t-----")
	for _, item := range result.Results {
		fmt.Fprintf(w, "%s\t%s\n", item.ID, item.Title)
	}
	return w.Flush()
}

func printJSON(result *service.SearchResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
