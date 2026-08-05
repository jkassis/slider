package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/gin-gonic/gin"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xeipuuv/gojsonschema"

	_ "github.com/frb/tpm/slider/statik"
)

var rootCmd = &cobra.Command{
	Use:   "slider",
	Short: "Render slideshow-style documents as web pages or PDFs",
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start web server to render documents",
	Run:   runServer,
}

var topdfCmd = &cobra.Command{
	Use:   "topdf",
	Short: "Render document to PDF file",
	Run:   runToPDF,
}

var setupCmd = &cobra.Command{
	Use:   "setup [directory]",
	Short: "Create example slider project structure",
	Long:  "Setup creates a new slider project with example templates, schemas, and documents. Defaults to current directory if no path specified.",
	Args:  cobra.MaximumNArgs(1),
	Run:   runSetup,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().String("dir", ".", "Data directory containing template/ and deck/ folders")
	rootCmd.PersistentFlags().String("deck", "", "Document to render (defaults to default.json, index.json, home.json, or root.json)")
	viper.BindPFlag("dir", rootCmd.PersistentFlags().Lookup("dir"))
	viper.BindPFlag("deck", rootCmd.PersistentFlags().Lookup("deck"))

	// Root-level flags

	// topdf command flags
	topdfCmd.Flags().String("output", "", "Output PDF path (defaults to pdf/ folder in data directory)")
	viper.BindPFlag("output", topdfCmd.Flags().Lookup("output"))

	// Add subcommands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(topdfCmd)
	rootCmd.AddCommand(setupCmd)
}

func initConfig() {
	viper.AutomaticEnv()
}

type Workstream struct {
	Name            string   `json:"name"`
	Accomplishments []string `json:"accomplishments"`
	NextSteps       []string `json:"nextSteps"`
}

type PageData struct {
	ProgramName    string       `json:"programName"`
	ProgramIconURL string       `json:"programIconUrl"`
	WeekEnding     string       `json:"weekEnding"`
	PageNumber     int          `json:"pageNumber"`
	TotalPages     int          `json:"-"`
	Workstreams    []Workstream `json:"workstreams"`
}

type Page struct {
	Template string   `json:"template"`
	Data     PageData `json:"data"`
}

type Document struct {
	Schema string `json:"schema"`
	Pages  []Page `json:"pages"`
}

func runServer(cmd *cobra.Command, args []string) {
	dataDir := viper.GetString("dir")
	if !filepath.IsAbs(dataDir) {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get working directory: %v\n", err)
			os.Exit(1)
		}
		dataDir = filepath.Join(wd, dataDir)
	}

	templatesPath := filepath.Join(dataDir, "template", "*")

	// Determine which document to load
	var dataPath string
	deckName := viper.GetString("deck")
	if deckName != "" {
		dataPath = filepath.Join(dataDir, "deck", deckName)
	} else {
		// Try default names in order
		defaultNames := []string{"default.json", "index.json", "home.json", "root.json"}
		found := false
		for _, name := range defaultNames {
			candidate := filepath.Join(dataDir, "deck", name)
			if _, err := os.Stat(candidate); err == nil {
				dataPath = candidate
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "Error: No document specified and no default found.\n")
			fmt.Fprintf(os.Stderr, "Tried the following files in %s:\n", filepath.Join(dataDir, "deck"))
			for _, name := range defaultNames {
				fmt.Fprintf(os.Stderr, "  - %s\n", name)
			}
			fmt.Fprintf(os.Stderr, "\nSpecify a document with --deck <filename>\n")
			os.Exit(1)
		}
	}

	// Load JSON data
	dataFile, err := os.ReadFile(dataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read data file: %v\n", err)
		os.Exit(1)
	}

	var doc Document
	if err := json.Unmarshal(dataFile, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse JSON: %v\n", err)
		os.Exit(1)
	}

	// Validate against schema if specified
	if doc.Schema != "" {
		schemaPath := filepath.Join(dataDir, "template", doc.Schema)
		schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)
		documentLoader := gojsonschema.NewGoLoader(doc)

		result, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load schema: %v\n", err)
			os.Exit(1)
		}

		if !result.Valid() {
			fmt.Fprintf(os.Stderr, "Document validation failed:\n")
			for _, desc := range result.Errors() {
				fmt.Fprintf(os.Stderr, "  - %s\n", desc)
			}
			os.Exit(1)
		}
	}

	router := gin.Default()
	router.SetFuncMap(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"lt":  func(a, b int) bool { return a < b },
		"gt":  func(a, b int) bool { return a > b },
	})
	router.LoadHTMLGlob(templatesPath)
	router.Static("/img", filepath.Join(dataDir, "img"))

	// Inject TotalPages into each page data
	totalPages := len(doc.Pages)
	for i := range doc.Pages {
		doc.Pages[i].Data.TotalPages = totalPages
	}

	port := rand.Intn(65535-49152) + 49152

	router.GET("/", func(c *gin.Context) {
		if len(doc.Pages) > 0 {
			c.HTML(http.StatusOK, doc.Pages[0].Template, doc.Pages[0].Data)
		} else {
			c.String(http.StatusNotFound, "No pages available")
		}
	})

	router.GET("/page/:num", func(c *gin.Context) {
		pageNum := c.Param("num")
		var pageIndex int
		fmt.Sscanf(pageNum, "%d", &pageIndex)
		pageIndex-- // Convert to 0-based index

		if pageIndex >= 0 && pageIndex < len(doc.Pages) {
			c.HTML(http.StatusOK, doc.Pages[pageIndex].Template, doc.Pages[pageIndex].Data)
		} else {
			c.String(http.StatusNotFound, "Page not found")
		}
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	go func() {
		url := fmt.Sprintf("http://localhost:%d", port)
		fmt.Printf("Server starting on port %d\n", port)
		fmt.Printf("Open in browser: \x1b]8;;%s\x07%s\x1b]8;;\x07\n", url, url)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server forced to shutdown: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Server exited")
}

func renderToPDF(dataDir string, doc Document, outputPath string) error {
	templatesPath := filepath.Join(dataDir, "template", "*")

	// Create a temporary HTTP server
	port := rand.Intn(65535-49152) + 49152
	router := gin.New()
	router.SetFuncMap(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"lt":  func(a, b int) bool { return a < b },
		"gt":  func(a, b int) bool { return a > b },
	})
	router.LoadHTMLGlob(templatesPath)
	router.Static("/img", filepath.Join(dataDir, "img"))

	router.GET("/page/:num", func(c *gin.Context) {
		pageNum := c.Param("num")
		var pageIndex int
		fmt.Sscanf(pageNum, "%d", &pageIndex)
		pageIndex--

		if pageIndex >= 0 && pageIndex < len(doc.Pages) {
			c.HTML(http.StatusOK, doc.Pages[pageIndex].Template, doc.Pages[pageIndex].Data)
		} else {
			c.String(http.StatusNotFound, "Page not found")
		}
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Create chromedp context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Collect PDF bytes for each page
	var pdfBuffers [][]byte
	for i := 1; i <= len(doc.Pages); i++ {
		url := fmt.Sprintf("http://localhost:%d/page/%d", port, i)
		var buf []byte

		if err := chromedp.Run(ctx,
			chromedp.Navigate(url),
			chromedp.WaitReady("body"),
			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				buf, _, err = page.PrintToPDF().
					WithLandscape(true).
					WithPrintBackground(true).
					WithPreferCSSPageSize(false).
					WithPaperWidth(11).
					WithPaperHeight(8.5).
					Do(ctx)
				return err
			}),
		); err != nil {
			return fmt.Errorf("failed to render page %d: %w", i, err)
		}
		pdfBuffers = append(pdfBuffers, buf)
	}

	// Write individual page PDFs to temp files and merge
	if len(pdfBuffers) == 0 {
		return fmt.Errorf("no pages rendered")
	}

	if len(pdfBuffers) == 1 {
		// Single page - just write directly
		if err := os.WriteFile(outputPath, pdfBuffers[0], 0644); err != nil {
			return fmt.Errorf("failed to write PDF: %w", err)
		}
		return nil
	}

	// Multiple pages - write temp files and merge
	tempDir, err := os.MkdirTemp("", "slider-pdf-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var tempFiles []string
	for i, buf := range pdfBuffers {
		tempFile := filepath.Join(tempDir, fmt.Sprintf("page-%d.pdf", i+1))
		if err := os.WriteFile(tempFile, buf, 0644); err != nil {
			return fmt.Errorf("failed to write temp PDF page %d: %w", i+1, err)
		}
		tempFiles = append(tempFiles, tempFile)
	}

	// Merge all PDFs
	if err := api.MergeCreateFile(tempFiles, outputPath, false, nil); err != nil {
		return fmt.Errorf("failed to merge PDFs: %w", err)
	}

	return nil
}

func runToPDF(cmd *cobra.Command, args []string) {
	dataDir := viper.GetString("dir")
	if !filepath.IsAbs(dataDir) {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get working directory: %v\n", err)
			os.Exit(1)
		}
		dataDir = filepath.Join(wd, dataDir)
	}


	// Determine which document to load
	var dataPath string
	deckName := viper.GetString("deck")
	if deckName != "" {
		dataPath = filepath.Join(dataDir, "deck", deckName)
	} else {
		// Try default names in order
		defaultNames := []string{"default.json", "index.json", "home.json", "root.json"}
		found := false
		for _, name := range defaultNames {
			candidate := filepath.Join(dataDir, "deck", name)
			if _, err := os.Stat(candidate); err == nil {
				dataPath = candidate
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "Error: No document specified and no default found.\n")
			fmt.Fprintf(os.Stderr, "Tried the following files in %s:\n", filepath.Join(dataDir, "deck"))
			for _, name := range defaultNames {
				fmt.Fprintf(os.Stderr, "  - %s\n", name)
			}
			fmt.Fprintf(os.Stderr, "\nSpecify a document with --deck <filename>\n")
			os.Exit(1)
		}
	}

	// Load JSON data
	dataFile, err := os.ReadFile(dataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read data file: %v\n", err)
		os.Exit(1)
	}

	var doc Document
	if err := json.Unmarshal(dataFile, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse JSON: %v\n", err)
		os.Exit(1)
	}

	// Validate against schema if specified
	if doc.Schema != "" {
		schemaPath := filepath.Join(dataDir, "template", doc.Schema)
		schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)
		documentLoader := gojsonschema.NewGoLoader(doc)

		result, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load schema: %v\n", err)
			os.Exit(1)
		}

		if !result.Valid() {
			fmt.Fprintf(os.Stderr, "Document validation failed:\n")
			for _, desc := range result.Errors() {
				fmt.Fprintf(os.Stderr, "  - %s\n", desc)
			}
			os.Exit(1)
		}
	}

	// Inject TotalPages into each page data
	totalPages := len(doc.Pages)
	for i := range doc.Pages {
		doc.Pages[i].Data.TotalPages = totalPages
	}

	pdfPath := viper.GetString("output")

	// If no explicit path given, default to pdf folder in data dir
	if pdfPath == "" {
		pdfDir := filepath.Join(dataDir, "pdf")
		if err := os.MkdirAll(pdfDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create pdf directory: %v\n", err)
			os.Exit(1)
		}
		// Use doc filename as base
		docBase := filepath.Base(dataPath)
		docBase = docBase[:len(docBase)-len(filepath.Ext(docBase))]
		pdfPath = filepath.Join(pdfDir, docBase+".pdf")
	}

	if err := renderToPDF(dataDir, doc, pdfPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render PDF: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("PDF generated: %s\n", pdfPath)
}

func runSetup(cmd *cobra.Command, args []string) {
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	// Get absolute path
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resolve target directory: %v\n", err)
		os.Exit(1)
	}

	// Open embedded filesystem
	statikFS, err := fs.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open embedded filesystem: %v\n", err)
		os.Exit(1)
	}

	// Walk the embedded filesystem and extract files
	err = fs.Walk(statikFS, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .DS_Store files
		if strings.HasSuffix(path, ".DS_Store") {
			return nil
		}

		// Skip the pdf folder - it's generated
		if strings.Contains(path, "/pdf/") || path == "/pdf" {
			return nil
		}

		targetPath := filepath.Join(absTarget, path)

		if info.IsDir() {
			// Create directory
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			fmt.Printf("Created directory: %s\n", targetPath)
		} else {
			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
			}

			// Read file from embedded FS
			file, err := statikFS.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open embedded file %s: %w", path, err)
			}
			defer file.Close()

			// Create target file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}
			defer outFile.Close()

			// Copy contents
			if _, err := io.Copy(outFile, file); err != nil {
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}

			fmt.Printf("Created file: %s\n", targetPath)
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Scaffold failed: %v\n", err)
		os.Exit(1)
	}

	// Create pdf directory
	pdfDir := filepath.Join(absTarget, "pdf")
	if err := os.MkdirAll(pdfDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create pdf directory: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created directory: %s\n", pdfDir)

	fmt.Printf("\n✓ Setup complete in %s\n", absTarget)
	fmt.Printf("Run 'slider --dir %s' to start the server\n", absTarget)
}

func main() {

	// Show help if no subcommand provided
	if len(os.Args) == 1 {
		rootCmd.Help()
		return
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
