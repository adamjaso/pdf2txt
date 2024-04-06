package extract

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	isXYPattern        = regexp.MustCompile(".* Tm$")
	isNLPattern        = regexp.MustCompile(`\s*ET\s*`)
	isTextPattern      = regexp.MustCompile(".*T[Jj]$")
	ignorePattern      = regexp.MustCompile(`.* Tf\s*$`)
	textTJSqParPattern = regexp.MustCompile(`^\s*\[(\(.+\))\]\s+TJ\s*$`)
	textTjParPattern   = regexp.MustCompile(`^\s*(\(.+\))Tj$`)
	textBytesPattern   = regexp.MustCompile(`^\s*<([0-9A-F]+)>\s*Tj\s*$`)
)

type (
	Format string
)

const (
	FormatPositioned Format = "positioned"
	FormatBytes      Format = "bytes"
	FormatAuto       Format = "auto"
)

func DetectFormat(pages ...*Page) Format {
	for _, p := range pages {
		for _, line := range p.Lines {
			if textBytesPattern.MatchString(line) {
				return FormatBytes
			}
		}
	}
	return FormatPositioned
}

func GetValidString(s string) string {
	for _, c := range s {
		if c > 127 {
			return ""
		}
	}
	return s
}

type (
	RenderConfig struct {
		Width         int64
		Height        int64
		Verbose       bool
		Fit           bool
		VerticalSpace bool
		ParseBytes    bool
	}
	Element struct {
		X        int64   `json:"x,omitempty"`
		Y        int64   `json:"y,omitempty"`
		X0       float64 `json:"x0"`
		Y0       float64 `json:"y0"`
		Text     string  `json:"text"`
		XYLine   string  `json:"xy_line,omitempty"`
		TextLine string  `json:"text_line,omitempty"`
	}
	PageElements struct {
		page             *Page      `json:"-"`
		Number           int        `json:"number"`
		MX               float64    `json:"mx"`
		MY               float64    `json:"my"`
		Elements         []*Element `json:"elements"`
		xyLine, textLine string
		x, y             float64
		table            map[float64]map[float64]*Element
	}
)

func (e *Element) Less(e2 *Element) bool {
	return e.Y < e2.Y || (e.Y == e2.Y && e.X < e2.X)
}

func (t *PageElements) parseXY(line string) error {
	if parts := strings.Split(line, " "); len(parts) < 7 {
		return fmt.Errorf("parseposition: line is less than 7 parts %q", line)
	} else if parts[6] != "Tm" {
		return fmt.Errorf("parseposition: not a position line %v", parts)
	} else if x, err := strconv.ParseFloat(parts[4], 64); err != nil {
		return fmt.Errorf("parseposition: parsefloat x %v", err)
	} else if y, err := strconv.ParseFloat(parts[5], 64); err != nil {
		return fmt.Errorf("parseposition: parsefloat y %v", err)
	} else {
		t.x = x
		t.y = y
		t.MX = math.Max(t.MX, x)
		t.MY = math.Max(t.MY, y)
		return nil
	}
}

func (t *PageElements) parseText(line string) (string, error) {
	if matches := textTJSqParPattern.FindAllStringSubmatch(line, -1); matches != nil {
		if text := strings.TrimSpace(GetValidString(matches[0][1])); text != "" {
			//t.addXYText(t.x, t.y, text)
			return parseTJLine(text), nil
		}
	} else if matches := textTjParPattern.FindAllStringSubmatch(line, -1); matches != nil {
		if text := strings.TrimSpace(GetValidString(matches[0][1])); text != "" {
			//t.addXYText(t.x, t.y, text)
			return parseTJLine(text), nil
		}
	} else if matches := textBytesPattern.FindAllStringSubmatch(line, -1); matches != nil {
		if text := matches[0][1]; text != "" {
			if textb, err := hex.DecodeString(text); err != nil {
				return "", fmt.Errorf("parsetext: hex decode %v", err)
			} else if text := GetValidString(string(textb)); text != "" {
				//t.addXYText(t.x, t.y, text)
				return text, nil
			}
		}
	}
	return "", fmt.Errorf("parsetext: no matches for %q", line)
}

func parseTJLine(line string) string {
	text := &bytes.Buffer{}
	inside := false
	pc := rune(0)
	for _, c := range line {
		if c == '(' && pc != '\\' {
			inside = true
		} else if c == ')' && pc != '\\' {
			inside = false
		} else if inside {
			text.WriteRune(c)
		}
		pc = c
	}
	return text.String()
}

func (t *PageElements) addXYText(x, y float64, text string) *Element {
	if _, ok := t.table[y]; !ok {
		t.table[y] = map[float64]*Element{}
	}
	el := &Element{
		X0:       x,
		Y0:       y,
		Text:     strings.ReplaceAll(text, "\\", ""),
		XYLine:   t.xyLine,
		TextLine: t.textLine,
	}
	t.table[y][x] = el
	return el
}

func ParsePageElements(p *Page, verbose bool) *PageElements {
	t := &PageElements{
		page:     p,
		Number:   p.Number,
		Elements: []*Element{},
		MX:       0,
		MY:       0,
		table:    map[float64]map[float64]*Element{},
		x:        0,
		y:        0,
	}
	for lnum, line := range p.Lines {
		if isXYPattern.MatchString(line) {
			t.xyLine = line
			if err := t.parseXY(line); err != nil {
				if verbose {
					log.Println(err)
				}
			}
			continue
		} else if isTextPattern.MatchString(line) {
			t.textLine = line
			if text, err := t.parseText(line); err != nil {
				if verbose {
					log.Println(err)
				}
			} else {
				t.addXYText(t.x, t.y, text)
			}
		} else if ignorePattern.MatchString(line) {
			if verbose {
				log.Printf("skipping line %05d: %q", lnum, line)
			}
			continue
		}
		t.x = 0
		t.y = 0
		t.xyLine = ""
		t.textLine = ""
	}
	for _, row := range t.table {
		for _, el := range row {
			t.Elements = append(t.Elements, el)
		}
	}
	return t
}

func ParsePages(pages []*Page, verbose bool) []*PageElements {
	elements := []*PageElements{}
	for _, p := range pages {
		elements = append(elements, ParsePageElements(p, verbose))
	}
	return elements
}

func ParsePageElementsBytes(p *Page, verbose bool) *PageElements {
	t := &PageElements{
		page:     p,
		Number:   p.Number,
		Elements: []*Element{},
	}
	x, y := int64(0), int64(0)
	for _, line := range p.Lines {
		if isNLPattern.MatchString(line) {
			x = 0
			y += 1
			continue
		} else if textBytesPattern.MatchString(line) {
			t.textLine = line
			if text, err := t.parseText(line); err != nil {
				if verbose {
					log.Println(err)
				}
			} else {
				t.Elements = append(t.Elements, &Element{
					X:    x,
					Y:    y,
					Text: text,
				})
				x += int64(len(text))
			}
		}
	}
	return t
}

func ParsePagesBytes(pages []*Page, verbose bool) []*PageElements {
	pe := []*PageElements{}
	for _, p := range pages {
		pe = append(pe, ParsePageElementsBytes(p, verbose))
	}
	return pe
}

func (rc *RenderConfig) Calculate(pel []*PageElements) {
	width := float64(rc.Width)
	height := float64(rc.Height)
	for _, p := range pel {
		if p.table == nil {
			continue
		}
		for _, e := range p.Elements {
			xpos := float64(int64(e.X0/p.page.Width*width) + 1)
			if rc.Fit {
				xendpos := width - float64(len(e.Text))
				xpos = math.Min(xpos, xendpos)
			}
			e.X = int64(xpos)
			e.Y = int64((p.page.Height - e.Y0) / p.page.Height * height)
		}
		sort.SliceStable(p.Elements, func(i, j int) bool {
			return p.Elements[i].Less(p.Elements[j])
		})
	}
}

func (rc *RenderConfig) Render(pages []*PageElements, out io.Writer) {
	for _, p := range pages {
		var prefix, suffix string
		if rc.Verbose {
			prefix, suffix = fmt.Sprintf("p%03d: ", p.Number), "$"
		} else {
			prefix, suffix = "", ""
		}
		x, y := int64(0), int64(0)
		for _, el := range p.Elements {
			if el.Y > y {
				if x < rc.Width {
					fmt.Fprint(out, strings.Repeat(" ", int(rc.Width-x)))
				}
				if rc.Verbose {
					fmt.Fprint(out, suffix)
				}
				fmt.Fprint(out, "\n")
				if rc.VerticalSpace {
					for i := int64(0); i < el.Y-y-1; i += 1 {
						fmt.Fprintf(out, prefix+strings.Repeat(" ", int(rc.Width))+suffix+"\n")
					}
				}
				x = 0
			}
			if x == 0 && rc.Verbose {
				fmt.Fprint(out, prefix)
			}
			if el.X > x {
				fmt.Fprint(out, strings.Repeat(" ", int(el.X-x)))
			}
			if len(el.Text) > 0 {
				fmt.Fprint(out, el.Text)
			}
			x = el.X + int64(len(el.Text))
			y = el.Y
		}
		fmt.Fprint(out, "\n")
	}
}
