package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/adamjaso/pdf2txt/extract"
)

type (
	App struct {
		Positioned, Bytes, Raw, Lines, Elements bool
		RenderConfig                            *extract.RenderConfig
	}
)

func convertPDF(app App, filename string, input io.ReadSeeker) {
	if !app.Raw {
		pages, err := extract.ExtractPages(input, true)
		if err != nil {
			log.Printf("extract err: %v", err)
			return
		} else if app.Lines {
			if err := json.NewEncoder(os.Stdout).Encode(&pages); err != nil {
				log.Printf("lines err: %v", err)
			}
			return
		}
		if !app.Bytes && !app.Positioned {
			format := extract.DetectFormat(pages...)
			app.Bytes = format == extract.FormatBytes
			app.Positioned = format == extract.FormatPositioned
		}
		var pel []*extract.PageElements
		if app.Bytes {
			pel = extract.ParsePagesBytes(pages, app.RenderConfig.Verbose)
		} else if app.Positioned {
			pel = extract.ParsePages(pages, app.RenderConfig.Verbose)
		} else {
			log.Printf("unable to detect format for %q", filename)
			return
		}
		app.RenderConfig.Calculate(pel)
		if app.Elements {
			if err := json.NewEncoder(os.Stdout).Encode(&pel); err != nil {
				log.Printf("lines err: %v", err)
				return
			}
		} else {
			app.RenderConfig.Render(pel, os.Stdout)
		}
	} else if _, err := extract.ExtractText(input, os.Stdout); err != nil {
		log.Printf("err: %v", err)
	}
}

func getFiles() ([]string, error) {
	globs := make([]string, flag.NArg())
	for i := range globs {
		globs[i] = flag.Arg(i)
	}
	results := []string{}
	for _, gpath := range globs {
		if gpath == "-" {
			return nil, nil
		}
		paths, err := filepath.Glob(gpath)
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			results = append(results, path)
		}
	}
	return results, nil
}

func main() {
	app := App{RenderConfig: &extract.RenderConfig{}}
	flag.BoolVar(&app.Raw, "or", false, "Output raw lines")
	flag.BoolVar(&app.Lines, "ol", false, "JSON output raw lines")
	flag.BoolVar(&app.Elements, "oe", false, "JSON output parsed elements")
	flag.BoolVar(&app.Positioned, "fp", false, "Parse as positioned text i.e. 1 0 0 1 XPos YPos Tm...[(Text here)] TJ...(Text here) Tj")
	flag.BoolVar(&app.Bytes, "fb", false, `Parse as byte strings i.e. <01234567890ABCDEF>Tj...ET -> bytestring(01, 23, ..., EF)..."\n"`)
	flag.BoolVar(&app.RenderConfig.Verbose, "v", false, "Verbose output")
	flag.BoolVar(&app.RenderConfig.Fit, "fit", false, "Fit output to constraints")
	flag.BoolVar(&app.RenderConfig.VerticalSpace, "verticalspace", false, "Show empty vertical space (i.e. on blank pages)")
	flag.Int64Var(&app.RenderConfig.Width, "w", 160, "Output elements width")
	flag.Int64Var(&app.RenderConfig.Height, "h", 120, "Output elements height")
	flag.Parse()
	log.SetOutput(os.Stderr)
	files, err := getFiles()
	if err != nil {
		log.Println(err)
		return
	} else if files == nil {
		convertPDF(app, "-", os.Stdin)
		return
	}
	for _, fname := range files {
		input, err := os.Open(fname)
		if err != nil {
			log.Printf("failed to open %s: %v", fname, err)
			return
		}
		defer input.Close()
		convertPDF(app, fname, input)
	}
}
