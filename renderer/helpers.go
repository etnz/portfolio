package renderer

import (
	"bytes"
	"io"
)

// SectionPrinter is a helper to conditionally print a header and a footer for a section
// only if content is actually written to it.
type SectionPrinter struct {
	headerFunc       func(io.Writer)
	footerFunc       func(io.Writer)
	hasPrintedHeader bool
}

// Header creates a new SectionPrinter and sets the function that will be called to print the section header.
// Deprecated: use ConditionalBlock instead.
func Header(f func(io.Writer)) *SectionPrinter {
	return &SectionPrinter{headerFunc: f}
}

// Footer sets the function that will be called to print the section footer.
func (p *SectionPrinter) Footer(f func(io.Writer)) *SectionPrinter {
	p.footerFunc = f
	return p
}

// PrintHeader prints the section header, but only on the first call.
// Subsequent calls do nothing. It should be called just before printing the first row.
func (p *SectionPrinter) PrintHeader(w io.Writer) {
	if p.hasPrintedHeader {
		return
	}
	p.hasPrintedHeader = true
	if p.headerFunc != nil {
		p.headerFunc(w)
	}
}

// PrintFooter prints the section footer, but only if the header was ever printed.
// It should be called after the loop that prints the rows.
func (p *SectionPrinter) PrintFooter(w io.Writer) {
	if p.hasPrintedHeader && p.footerFunc != nil {
		p.footerFunc(w)
	}
}

// ConditionalBlock let you fully write a block and decide at the end to print it or not.
// If the block function returns true, the content is printed to w, otherwise it is discarded.
func ConditionalBlock(w io.Writer, block func(io.Writer) bool) {
	bw := &bytes.Buffer{}
	if block(bw) {
		io.Copy(w, bw)
	}
}
