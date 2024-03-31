# pdf2txt

A basic demonstration of how to use the excellent [pdfcpu library](https://github.com/pdfcpu/pdfcpu).

## Why?

`pdfcpu` is an excellent tool for extracting the text content of PDF documents. However, it's CLI
interface is limited to specifying a PDF file and an output directory where each extracted bit
of content is saved in a separate file. When writing small scripts to parse PDFs it's useful to
just pass a PDF to STDIN and read parsed text output from STDOUT which may, in turn, be piped
into more commands that extract the desired information from the PDF text. For this purpose,
`pdfcpu` requires creating a number of unnecessary files, so this tool provides a way
to extract text from a PDF read from STDIN, and outputs the extracted text to STDOUT.

This tool is also provides a simplified function that may be imported into other projects for the purpose
of processing PDFs to text in memory.
