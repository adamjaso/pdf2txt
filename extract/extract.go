package extract

import (
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
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
