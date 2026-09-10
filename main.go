package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/gophernment/decky/internal/exporter"
	"github.com/gophernment/decky/internal/server"
)

//go:embed USAGE.md
var usageGuide string

var startServer = func(markdownFile, port string, stdout io.Writer) error {
	return server.Start(markdownFile, port, stdout)
}

var startExport = func(markdownFile, outputFile string, stdout io.Writer) error {
	return exporter.Export(markdownFile, outputFile, stdout)
}

var startExportHTML = func(markdownFile, outputFile string, stdout io.Writer) error {
	return exporter.ExportHTML(markdownFile, outputFile)
}

func main() {
	if err := run(os.Args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) < 2 {
		printUsage(stderr)
		return errors.New("missing command")
	}

	command := args[1]
	switch command {
	case "serve":
		serveCmd := flag.NewFlagSet("serve", flag.ContinueOnError)
		serveCmd.SetOutput(stderr)
		defaultPort := "3325"
		if envPort := os.Getenv("PORT"); envPort != "" {
			defaultPort = envPort
		}
		port := serveCmd.String("port", defaultPort, "Port to serve presentation on (defaults to $PORT env var, or 3325)")
		serveCmd.Usage = func() {
			fmt.Fprintln(serveCmd.Output(), "Usage: decky serve [-port 3325] <file.md>")
			fmt.Fprintln(serveCmd.Output(), "Options:")
			serveCmd.PrintDefaults()
		}
		if err := serveCmd.Parse(args[2:]); err != nil {
			return err
		}
		if serveCmd.NArg() < 1 {
			fmt.Fprintln(stderr, "Error: Markdown file path required for serve command")
			return errors.New("missing markdown file")
		}
		markdownFile := serveCmd.Arg(0)
		fmt.Fprintf(stdout, "Serving %s on port %s...\n", markdownFile, *port)
		if err := startServer(markdownFile, *port, stdout); err != nil {
			return err
		}
		return nil

	case "export":
		exportCmd := flag.NewFlagSet("export", flag.ContinueOnError)
		exportCmd.SetOutput(stderr)
		output := exportCmd.String("o", "presentation.pdf", "Output PDF file path")
		exportCmd.Usage = func() {
			fmt.Fprintln(exportCmd.Output(), "Usage: decky export [-o output.pdf] <file.md>")
			fmt.Fprintln(exportCmd.Output(), "Options:")
			exportCmd.PrintDefaults()
		}
		if err := exportCmd.Parse(args[2:]); err != nil {
			return err
		}
		if exportCmd.NArg() < 1 {
			fmt.Fprintln(stderr, "Error: Markdown file path required for export command")
			return errors.New("missing markdown file")
		}
		markdownFile := exportCmd.Arg(0)
		fmt.Fprintf(stdout, "Exporting %s to %s...\n", markdownFile, *output)
		if err := startExport(markdownFile, *output, stdout); err != nil {
			return err
		}
		return nil

	case "html":
		htmlCmd := flag.NewFlagSet("html", flag.ContinueOnError)
		htmlCmd.SetOutput(stderr)
		output := htmlCmd.String("o", "presentation.html", "Output HTML file path")
		htmlCmd.Usage = func() {
			fmt.Fprintln(htmlCmd.Output(), "Usage: decky html [-o output.html] <file.md>")
			fmt.Fprintln(htmlCmd.Output(), "Options:")
			htmlCmd.PrintDefaults()
		}
		if err := htmlCmd.Parse(args[2:]); err != nil {
			return err
		}
		if htmlCmd.NArg() < 1 {
			fmt.Fprintln(stderr, "Error: Markdown file path required for html command")
			return errors.New("missing markdown file")
		}
		markdownFile := htmlCmd.Arg(0)
		fmt.Fprintf(stdout, "Exporting %s to %s...\n", markdownFile, *output)
		if err := startExportHTML(markdownFile, *output, stdout); err != nil {
			return err
		}
		return nil

	case "usage":
		usageCmd := flag.NewFlagSet("usage", flag.ContinueOnError)
		usageCmd.SetOutput(stderr)
		output := usageCmd.String("o", "USAGE.md", "Output file path")
		usageCmd.Usage = func() {
			fmt.Fprintln(usageCmd.Output(), "Usage: decky usage [-o USAGE.md]")
			fmt.Fprintln(usageCmd.Output(), "Options:")
			usageCmd.PrintDefaults()
		}
		if err := usageCmd.Parse(args[2:]); err != nil {
			return err
		}
		if err := os.WriteFile(*output, []byte(usageGuide), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Wrote %s — run \"decky serve %s\" to try it.\n", *output, *output)
		return nil

	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command: %s", command)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: decky <command> [arguments]")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  serve [-port 3325] <file.md>  Start the presentation server")
	fmt.Fprintln(w, "  export [-o output.pdf] <file.md>  Export to a single PDF file")
	fmt.Fprintln(w, "  html [-o output.html] <file.md>  Export to a single self-contained HTML file")
	fmt.Fprintln(w, "  usage [-o USAGE.md]  Write the embedded usage guide to a file")
}
