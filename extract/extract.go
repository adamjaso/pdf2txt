package extract

import (
	"bufio"
	"bytes"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type (
	Page struct {
		Number int      `json:"page"`
		Lines  []string `json:"lines,omitempty"`
		Text   string   `json:"text,omitempty"`
	}
)

func ExtractText(rs io.ReadSeeker, output io.Writer) (*model.Context, error) {
	conf := model.NewDefaultConfiguration()
	conf.Cmd = model.EXTRACTCONTENT
	ctx, err := api.ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, err
	}
	for p := 1; p <= ctx.PageCount; p += 1 {
		if r, err := pdfcpu.ExtractPageContent(ctx, p); err != nil {
			return nil, err
		} else if r == nil {
			continue
		} else if _, err := io.Copy(output, r); err != nil {
			return nil, err
		}
	}
	return ctx, nil
}

func ExtractPages(rs io.ReadSeeker, parseLines bool) ([]*Page, error) {
	conf := model.NewDefaultConfiguration()
	conf.Cmd = model.EXTRACTCONTENT
	ctx, err := api.ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, err
	}
	pages := make([]*Page, ctx.PageCount)
	for p := 1; p <= ctx.PageCount; p += 1 {
		page := &Page{Number: p}
		text := &bytes.Buffer{}
		if r, err := pdfcpu.ExtractPageContent(ctx, p); err != nil {
			return nil, err
		} else if r == nil {
			continue
		} else if parseLines {
			page.Lines = []string{}
			lines := bufio.NewScanner(r)
			for lines.Scan() {
				page.Lines = append(page.Lines, lines.Text())
			}
			if err := lines.Err(); err != nil {
				return nil, err
			}
		} else if _, err := io.Copy(text, r); err != nil {
			return nil, err
		} else {
			page.Text = text.String()
		}
		pages[p-1] = page
	}
	return pages, nil
}
