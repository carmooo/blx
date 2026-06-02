package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	root.AddCommand(itemCmd())
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

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func itemCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "item <id>",
		Short: "Show full item details and holdings",
		Long:  "Show full item details and holdings.\n\nUse single quotes around the ID to avoid shell interpretation:\n  blx item '3100024~!29075~!1'",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			id := args[0]

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			client := ipac.NewClient()
			repo := ipac.NewRepository(client)

			item, err := repo.GetItem(ctx, id)
			if err != nil {
				return fmt.Errorf("get item: %w", err)
			}

			holdings, err := repo.GetHoldings(ctx, id)
			if err != nil {
				return fmt.Errorf("get holdings: %w", err)
			}
			item.Holdings = holdings

			if jsonOut {
				return printJSON(item)
			}
			return printItem(item)
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func printItem(item *service.Item) error {
	fmt.Printf("Title:     %s\n", item.Title)

	if len(item.Authors) > 0 {
		var names []string
		for _, a := range item.Authors {
			names = append(names, a.Name)
		}
		fmt.Printf("Authors:   %s\n", strings.Join(names, " ; "))
	}

	if item.Publisher != "" || item.Year != "" {
		pub := item.Publisher
		if item.Year != "" {
			if pub != "" {
				pub += ", " + item.Year
			} else {
				pub = item.Year
			}
		}
		fmt.Printf("Publisher: %s\n", pub)
	}

	if item.ISBN != "" {
		fmt.Printf("ISBN:      %s\n", item.ISBN)
	}
	if item.Language != "" {
		fmt.Printf("Language:  %s\n", item.Language)
	}
	if item.Edition != "" {
		fmt.Printf("Edition:   %s\n", item.Edition)
	}
	if item.Physical != "" {
		fmt.Printf("Physical:  %s\n", item.Physical)
	}
	if len(item.Subjects) > 0 {
		fmt.Printf("Subjects:  %s\n", strings.Join(item.Subjects, " ; "))
	}
	if item.Classification != "" {
		fmt.Printf("Class:     %s\n", item.Classification)
	}

	if len(item.Holdings) > 0 {
		fmt.Println()
		fmt.Println("Holdings:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "BRANCH\tCALL NUMBER\tSTATUS")
		fmt.Fprintln(w, "------\t-----------\t------")
		for _, h := range item.Holdings {
			fmt.Fprintf(w, "%s\t%s\t%s\n", h.Branch, h.CallNumber, h.Status)
		}
		return w.Flush()
	}

	return nil
}
