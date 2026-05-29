package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hd-health/hd-health/internal/platform"
	"github.com/hd-health/hd-health/internal/remediate"
	"github.com/hd-health/hd-health/internal/report"
	"github.com/hd-health/hd-health/internal/scan"
	"github.com/hd-health/hd-health/internal/store"
)

func main() {
	os.Exit(run())
}

func run() int {
	dbPath := flag.String("db", "", "SQLite metrics database path")
	verbose := flag.Bool("verbose-paths", false, "Include full paths in profile output")
	format := flag.String("format", "text", "export format: json or csv")
	mount := flag.String("mount", "/", "mount point for remediate/explain")
	dryRun := flag.Bool("dry-run", true, "remediate: print commands only")
	apply := flag.Bool("apply", false, "remediate: run allowlisted low-risk commands")
	aggressive := flag.Bool("aggressive", false, "remediate: include high-risk steps")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"report"}
	}
	cmd := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	plat := platform.Current()
	var st *store.Store
	if *dbPath != "" || cmd == "report" || cmd == "forecast" || cmd == "export" || cmd == "agent-once" {
		var err error
		st, err = store.Open(*dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "store: %v\n", err)
			return 1
		}
		defer st.Close()
	}

	switch cmd {
	case "scan":
		res, err := scan.Quick(ctx, plat, *mount)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scan: %v\n", err)
			return 1
		}
		fmt.Print(res.Volumes)
		if len(res.Suggestions) > 0 {
			fmt.Println("\nSuggested tools:")
			for _, s := range res.Suggestions {
				fmt.Println(" ", s)
			}
		}
		return 0

	case "report", "forecast":
		b := &report.Builder{Platform: plat, Store: st, VerbosePaths: *verbose}
		r, code, err := b.Build(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "report: %v\n", err)
			return 1
		}
		fmt.Print(report.FormatHuman(r))
		return code

	case "explain":
		m := *mount
		if len(args) >= 2 {
			m = args[1]
		}
		if !strings.HasPrefix(m, "/") {
			m = "/" + m
		}
		*mount = m
		b := &report.Builder{Platform: plat, Store: st, VerbosePaths: true}
		r, _, err := b.Build(ctx)
		if err != nil {
			return 1
		}
		fmt.Printf("Explain mount: %s\n\n", *mount)
		for _, p := range r.Profiles {
			fmt.Printf("Profile: %s (%.1f GB, confidence %.0f%%)\n", p.Name, float64(p.Bytes)/(1<<30), p.Confidence*100)
			for _, path := range p.Paths {
				fmt.Printf("  %s\n", path)
			}
		}
		dirs, err := plat.TopDirs(ctx, *mount, 2)
		if err == nil {
			fmt.Printf("\nDirectory tree (depth 2):\n")
			for i, d := range dirs {
				if i > 15 {
					break
				}
				fmt.Printf("  %.2f GB  %s\n", float64(d.Bytes)/(1<<30), d.Path)
			}
		}
		return 0

	case "remediate":
		opt := remediate.Options{
			Mount: *mount, DryRun: *dryRun && !*apply, Apply: *apply, Aggressive: *aggressive,
		}
		if *apply {
			opt.DryRun = false
		}
		res, err := remediate.Execute(ctx, opt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "remediate: %v\n", err)
			return 1
		}
		for _, line := range res.Run {
			fmt.Println(line)
		}
		for _, e := range res.Errors {
			fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		return 0

	case "export":
		b := &report.Builder{Platform: plat, Store: st, VerbosePaths: *verbose}
		r, code, err := b.Build(ctx)
		if err != nil {
			return 1
		}
		switch strings.ToLower(*format) {
		case "json":
			_ = report.WriteJSON(os.Stdout, r)
		case "csv":
			_ = report.WriteCSV(os.Stdout, r)
		default:
			fmt.Fprintf(os.Stderr, "unknown format %q\n", *format)
			return 1
		}
		return code

	case "agent-once":
		b := &report.Builder{Platform: plat, Store: st}
		_, code, err := b.Build(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "agent-once: %v\n", err)
			return 1
		}
		return code

	case "help", "-h", "--help":
		printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		printUsage()
		return 1
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `hd-health — storage health for macOS and Fedora

Usage:
  hd-health scan
  hd-health report
  hd-health forecast
  hd-health explain [mount]
  hd-health remediate [--mount /] [--dry-run|--apply] [--aggressive]
  hd-health export [--format json|csv]
  hd-health agent-once

Exit codes: 0=ok, 1=warning, 2=critical

`)
}
