package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/adamjaso/pdf2txt/extract"
)

func main() {
	var (
		fname string
		input io.ReadSeeker
		err   error
	)
	flag.StringVar(&fname, "f", "", "PDF file from which to extract text")
	flag.Parse()
	log.SetOutput(os.Stderr)
	if fname == "-" {
		input = os.Stdin
	} else if input, err = os.Open(fname); err != nil {
		log.Printf("failed to open %s: %v", fname, err)
		return
	}
	if _, err := extract.ExtractText(input, os.Stdout); err != nil {
		log.Printf("err: %v", err)
	}
}
