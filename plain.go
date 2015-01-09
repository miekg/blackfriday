// Plain text rendering backend

package mmark

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Plain is a type that implements the Renderer interface for Plain text output.
type Plain struct {
	flags    int    // HTML_* options
	closeTag string // how to end singleton tags: either " />\n" or ">\n"
	title    string // document title
	css      string // optional css file url (used with HTML_COMPLETE_PAGE)

	// store the IAL we see for this block element
	ial *InlineAttr

	parameters HtmlRendererParameters

	// table of contents data
	tocMarker    int
	headerCount  int
	currentLevel int
	toc          *bytes.Buffer
}

func PlainRenderer(flags int) Renderer {
	return &Plain{}
}

func (options *Plain) Flags() int { return options.flags }

func (options *Plain) TitleBlockTOML(out *bytes.Buffer, data *title) {}

func (options *Plain) Part(out *bytes.Buffer, text func() bool, id string) {}

func (options *Plain) Abstract(out *bytes.Buffer, text func() bool, id string) {
	// Create header with abstract
}

func (options *Plain) Header(out *bytes.Buffer, text func() bool, level int, id string) {
	marker := out.Len()
	doubleSpace(out)

	if id != "" {
		out.WriteString(fmt.Sprintf("<h%d id=\"%s\">", level, id))
	} else if options.flags&HTML_TOC != 0 {
		// headerCount is incremented in htmlTocHeader
		out.WriteString(fmt.Sprintf("<h%d id=\"toc_%d\">", level, options.headerCount))
	} else {
		out.WriteString(fmt.Sprintf("<h%d>", level))
	}

	tocMarker := out.Len()
	if !text() {
		out.Truncate(marker)
		return
	}

	// are we building a table of contents?
	if options.flags&HTML_TOC != 0 {
		options.TocHeaderWithAnchor(out.Bytes()[tocMarker:], level, id)
	}

	out.WriteString(fmt.Sprintf("</h%d>\n", level))
}

func (options *Plain) CommentHtml(out *bytes.Buffer, text []byte) {
	if options.flags&HTML_SKIP_HTML != 0 {
		return
	}

	doubleSpace(out)
	out.Write(text)
	out.WriteByte('\n')
}

func (options *Plain) BlockHtml(out *bytes.Buffer, text []byte) {
	if options.flags&HTML_SKIP_HTML != 0 {
		return
	}

	doubleSpace(out)
	out.Write(text)
	out.WriteByte('\n')
}

func (options *Plain) HRule(out *bytes.Buffer) {
	doubleSpace(out)
	out.WriteString("<hr")
	out.WriteString(options.closeTag)
}

func (options *Plain) BlockCode(out *bytes.Buffer, text []byte, lang string, caption []byte) {
	doubleSpace(out)

	// parse out the language names/classes
	count := 0
	for _, elt := range strings.Fields(lang) {
		if elt[0] == '.' {
			elt = elt[1:]
		}
		if len(elt) == 0 {
			continue
		}
		if count == 0 {
			out.WriteString("<pre><code class=\"language-")
		} else {
			out.WriteByte(' ')
		}
		attrEscape(out, []byte(elt))
		count++
	}

	if count == 0 {
		out.WriteString("<pre><code>")
	} else {
		out.WriteString("\">")
	}

	attrEscape(out, text)
	out.WriteString("</code></pre>\n")
}

func (options *Plain) BlockQuote(out *bytes.Buffer, text []byte, attribution []byte) {
	doubleSpace(out)
	out.WriteString("<blockquote>\n")
	out.Write(text)
	out.WriteString("</blockquote>\n")
}

func (options *Plain) Aside(out *bytes.Buffer, text []byte) {
	doubleSpace(out)
	out.WriteString("<blockquote>\n")
	out.Write(text)
	out.WriteString("</blockquote>\n")
}

func (options *Plain) Note(out *bytes.Buffer, text []byte) {
	doubleSpace(out)
	out.WriteString("<blockquote>\n")
	out.Write(text)
	out.WriteString("</blockquote>\n")
}

func (options *Plain) Table(out *bytes.Buffer, header []byte, body []byte, footer []byte, columnData []int, caption []byte) {
	doubleSpace(out)
	out.WriteString("<table>\n")
	if len(caption) > 0 {
		out.WriteString("<caption>\n")
		out.Write(caption)
		out.WriteString("\n</caption>\n")
	}
	out.WriteString("<thead>\n")
	out.Write(header)
	out.WriteString("</thead>\n\n<tbody>\n")
	out.Write(body)
	out.WriteString("</tbody>\n")
	if len(footer) > 0 {
		out.WriteString("<tfoot>\n")
		out.Write(footer)
		out.WriteString("</tfoot>\n")
	}
	out.WriteString("</table>\n")
}

func (options *Plain) TableRow(out *bytes.Buffer, text []byte) {
	doubleSpace(out)
	out.WriteString("<tr>\n")
	out.Write(text)
	out.WriteString("\n</tr>\n")
}

func (options *Plain) TableHeaderCell(out *bytes.Buffer, text []byte, align int) {
	doubleSpace(out)
	switch align {
	case TABLE_ALIGNMENT_LEFT:
		out.WriteString("<th align=\"left\">")
	case TABLE_ALIGNMENT_RIGHT:
		out.WriteString("<th align=\"right\">")
	case TABLE_ALIGNMENT_CENTER:
		out.WriteString("<th align=\"center\">")
	default:
		out.WriteString("<th>")
	}

	out.Write(text)
	out.WriteString("</th>")
}

func (options *Plain) TableCell(out *bytes.Buffer, text []byte, align int) {
	doubleSpace(out)
	switch align {
	case TABLE_ALIGNMENT_LEFT:
		out.WriteString("<td align=\"left\">")
	case TABLE_ALIGNMENT_RIGHT:
		out.WriteString("<td align=\"right\">")
	case TABLE_ALIGNMENT_CENTER:
		out.WriteString("<td align=\"center\">")
	default:
		out.WriteString("<td>")
	}

	out.Write(text)
	out.WriteString("</td>")
}

func (options *Plain) Footnotes(out *bytes.Buffer, text func() bool) {
	out.WriteString("<div class=\"footnotes\">\n")
	options.HRule(out)
	options.List(out, text, LIST_TYPE_ORDERED, 0, nil)
	out.WriteString("</div>\n")
}

func (options *Plain) FootnoteItem(out *bytes.Buffer, name, text []byte, flags int) {
	if flags&LIST_ITEM_CONTAINS_BLOCK != 0 || flags&LIST_ITEM_BEGINNING_OF_LIST != 0 {
		doubleSpace(out)
	}
	slug := slugify(name)
	out.WriteString(`<li id="`)
	out.WriteString(`fn:`)
	out.WriteString(options.parameters.FootnoteAnchorPrefix)
	out.Write(slug)
	out.WriteString(`">`)
	out.Write(text)
	if options.flags&HTML_FOOTNOTE_RETURN_LINKS != 0 {
		out.WriteString(` <a class="footnote-return" href="#`)
		out.WriteString(`fnref:`)
		out.WriteString(options.parameters.FootnoteAnchorPrefix)
		out.Write(slug)
		out.WriteString(`">`)
		out.WriteString(options.parameters.FootnoteReturnLinkContents)
		out.WriteString(`</a>`)
	}
	out.WriteString("</li>\n")
}

func (options *Plain) List(out *bytes.Buffer, text func() bool, flags, start int, group []byte) {
	marker := out.Len()
	doubleSpace(out)

	if flags&LIST_TYPE_ORDERED != 0 {
		switch {
		case flags&LIST_TYPE_ORDERED_ALPHA_LOWER != 0:
			out.WriteString("<ol type=\"a\">")
		case flags&LIST_TYPE_ORDERED_ALPHA_UPPER != 0:
			out.WriteString("<ol type=\"A\">")
		case flags&LIST_TYPE_ORDERED_ROMAN_LOWER != 0:
			out.WriteString("<ol type=\"i\">")
		case flags&LIST_TYPE_ORDERED_ROMAN_UPPER != 0:
			out.WriteString("<ol type=\"I\">")
		default:
			out.WriteString("<ol>")
		}
	} else {
		out.WriteString("<ul>")
	}
	if !text() {
		out.Truncate(marker)
		return
	}
	if flags&LIST_TYPE_ORDERED != 0 {
		out.WriteString("</ol>\n")
	} else {
		out.WriteString("</ul>\n")
	}
}

func (options *Plain) ListItem(out *bytes.Buffer, text []byte, flags int) {
	if flags&LIST_ITEM_CONTAINS_BLOCK != 0 || flags&LIST_ITEM_BEGINNING_OF_LIST != 0 {
		doubleSpace(out)
	}
	out.WriteString("<li>")
	out.Write(text)
	out.WriteString("</li>\n")
}

func (options *Plain) Example(out *bytes.Buffer, index int) {
	out.WriteByte('(')
	out.WriteString(strconv.Itoa(index))
	out.WriteByte(')')
}

func (options *Plain) Paragraph(out *bytes.Buffer, text func() bool, flags int) {
	marker := out.Len()
	doubleSpace(out)

	out.WriteString("<p>")
	if !text() {
		out.Truncate(marker)
		return
	}
	out.WriteString("</p>\n")
}

func (options *Plain) Math(out *bytes.Buffer, text []byte, display bool) {
	ial := options.InlineAttr()
	s := ial.String()
	if display {
		out.WriteString("<script " + s + "type=\"math/tex; mode=display\"> ")
	} else {
		out.WriteString("<script type=\"math/tex\"> ")

	}
	out.Write(text)
	out.WriteString("</script>")
}

func (options *Plain) AutoLink(out *bytes.Buffer, link []byte, kind int) {
	skipRanges := htmlEntity.FindAllIndex(link, -1)
	if options.flags&HTML_SAFELINK != 0 && !isSafeLink(link) && kind != LINK_TYPE_EMAIL {
		// mark it but don't link it if it is not a safe link
		out.WriteString("<tt>")
		entityEscapeWithSkip(out, link, skipRanges)
		out.WriteString("</tt>")
		return
	}

	out.WriteString("<a href=\"")
	if kind == LINK_TYPE_EMAIL {
		out.WriteString("mailto:")
	} else {
		options.maybeWriteAbsolutePrefix(out, link)
	}

	entityEscapeWithSkip(out, link, skipRanges)

	if options.flags&HTML_NOFOLLOW_LINKS != 0 && !isRelativeLink(link) {
		out.WriteString("\" rel=\"nofollow")
	}
	// blank target only add to external link
	if options.flags&HTML_HREF_TARGET_BLANK != 0 && !isRelativeLink(link) {
		out.WriteString("\" target=\"_blank")
	}

	out.WriteString("\">")

	// Pretty print: if we get an email address as
	// an actual URI, e.g. `mailto:foo@bar.com`, we don't
	// want to print the `mailto:` prefix
	switch {
	case bytes.HasPrefix(link, []byte("mailto://")):
		attrEscape(out, link[len("mailto://"):])
	case bytes.HasPrefix(link, []byte("mailto:")):
		attrEscape(out, link[len("mailto:"):])
	default:
		entityEscapeWithSkip(out, link, skipRanges)
	}

	out.WriteString("</a>")
}

func (options *Plain) CodeSpan(out *bytes.Buffer, text []byte) {
	out.WriteString("<code>")
	attrEscape(out, text)
	out.WriteString("</code>")
}

func (options *Plain) DoubleEmphasis(out *bytes.Buffer, text []byte) {
	out.WriteString("<strong>")
	out.Write(text)
	out.WriteString("</strong>")
}

func (options *Plain) Emphasis(out *bytes.Buffer, text []byte) {
	// TODO(miek): why is this check here?
	if len(text) == 0 {
		return
	}
	out.WriteString("<em>")
	out.Write(text)
	out.WriteString("</em>")
}

func (options *Plain) Subscript(out *bytes.Buffer, text []byte) {
	out.WriteString("<sub>")
	out.Write(text)
	out.WriteString("</sub>")
}

func (options *Plain) Superscript(out *bytes.Buffer, text []byte) {
	out.WriteString("<sup>")
	out.Write(text)
	out.WriteString("</sup>")
}

func (options *Plain) maybeWriteAbsolutePrefix(out *bytes.Buffer, link []byte) {
	if options.parameters.AbsolutePrefix != "" && isRelativeLink(link) {
		out.WriteString(options.parameters.AbsolutePrefix)
		if link[0] != '/' {
			out.WriteByte('/')
		}
	}
}

func (options *Plain) Image(out *bytes.Buffer, link []byte, title []byte, alt []byte) {
	if options.flags&HTML_SKIP_IMAGES != 0 {
		return
	}

	out.WriteString("<img src=\"")
	options.maybeWriteAbsolutePrefix(out, link)
	attrEscape(out, link)
	out.WriteString("\" alt=\"")
	if len(alt) > 0 {
		attrEscape(out, alt)
	}
	if len(title) > 0 {
		out.WriteString("\" title=\"")
		attrEscape(out, title)
	}

	out.WriteByte('"')
	out.WriteString(options.closeTag)
	return
}

func (options *Plain) LineBreak(out *bytes.Buffer) {
	out.WriteString("<br")
	out.WriteString(options.closeTag)
}

func (options *Plain) Link(out *bytes.Buffer, link []byte, title []byte, content []byte) {
	if options.flags&HTML_SKIP_LINKS != 0 {
		// write the link text out but don't link it, just mark it with typewriter font
		out.WriteString("<tt>")
		attrEscape(out, content)
		out.WriteString("</tt>")
		return
	}

	if options.flags&HTML_SAFELINK != 0 && !isSafeLink(link) {
		// write the link text out but don't link it, just mark it with typewriter font
		out.WriteString("<tt>")
		attrEscape(out, content)
		out.WriteString("</tt>")
		return
	}

	out.WriteString("<a href=\"")
	options.maybeWriteAbsolutePrefix(out, link)
	attrEscape(out, link)
	if len(title) > 0 {
		out.WriteString("\" title=\"")
		attrEscape(out, title)
	}
	if options.flags&HTML_NOFOLLOW_LINKS != 0 && !isRelativeLink(link) {
		out.WriteString("\" rel=\"nofollow")
	}
	// blank target only add to external link
	if options.flags&HTML_HREF_TARGET_BLANK != 0 && !isRelativeLink(link) {
		out.WriteString("\" target=\"_blank")
	}

	out.WriteString("\">")
	out.Write(content)
	out.WriteString("</a>")
	return
}

func (options *Plain) Abbreviation(out *bytes.Buffer, abbr, title []byte) {
	if len(title) == 0 {
		out.WriteString("<abbr>")
	} else {
		out.WriteString("<abbr title=\"")
		out.Write(title)
		out.WriteString("\">")
	}
	out.Write(abbr)
	out.WriteString("</abbr>")
}

func (options *Plain) RawHtmlTag(out *bytes.Buffer, text []byte) {
	if options.flags&HTML_SKIP_HTML != 0 {
		return
	}
	if options.flags&HTML_SKIP_STYLE != 0 && isHtmlTag(text, "style") {
		return
	}
	if options.flags&HTML_SKIP_LINKS != 0 && isHtmlTag(text, "a") {
		return
	}
	if options.flags&HTML_SKIP_IMAGES != 0 && isHtmlTag(text, "img") {
		return
	}
	out.Write(text)
}

func (options *Plain) TripleEmphasis(out *bytes.Buffer, text []byte) {
	out.WriteString("<strong><em>")
	out.Write(text)
	out.WriteString("</em></strong>")
}

func (options *Plain) StrikeThrough(out *bytes.Buffer, text []byte) {
	out.WriteString("<del>")
	out.Write(text)
	out.WriteString("</del>")
}

func (options *Plain) FootnoteRef(out *bytes.Buffer, ref []byte, id int) {
	slug := slugify(ref)
	out.WriteString(`<sup class="footnote-ref" id="`)
	out.WriteString(`fnref:`)
	out.WriteString(options.parameters.FootnoteAnchorPrefix)
	out.Write(slug)
	out.WriteString(`"><a rel="footnote" href="#`)
	out.WriteString(`fn:`)
	out.WriteString(options.parameters.FootnoteAnchorPrefix)
	out.Write(slug)
	out.WriteString(`">`)
	out.WriteString(strconv.Itoa(id))
	out.WriteString(`</a></sup>`)
}

func (options *Plain) Index(out *bytes.Buffer, primary, secondary []byte, prim bool) {}
func (options *Plain) Citation(out *bytes.Buffer, link, title []byte)                {}
func (options *Plain) References(out *bytes.Buffer, citations map[string]*citation)  {}
func (options *Plain) Entity(out *bytes.Buffer, entity []byte)                       { out.Write(entity) }

func (options *Plain) NormalText(out *bytes.Buffer, text []byte) {
	attrEscape(out, text)
}

func (options *Plain) DocumentHeader(out *bytes.Buffer, first bool) {
	if !first {
		return
	}
	if options.flags&HTML_COMPLETE_PAGE == 0 {
		return
	}

	ending := ""
	if options.flags&HTML_USE_XHTML != 0 {
		out.WriteString("<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\" ")
		out.WriteString("\"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">\n")
		out.WriteString("<html xmlns=\"http://www.w3.org/1999/xhtml\">\n")
		ending = " /"
	} else {
		out.WriteString("<!DOCTYPE html>\n")
		out.WriteString("<html>\n")
	}
	out.WriteString("<head>\n")
	out.WriteString("  <title>")
	options.NormalText(out, []byte(options.title))
	out.WriteString("</title>\n")
	out.WriteString("  <meta name=\"GENERATOR\" content=\"Blackfriday Markdown Processor v")
	out.WriteString(VERSION)
	out.WriteString("\"")
	out.WriteString(ending)
	out.WriteString(">\n")
	out.WriteString("  <meta charset=\"utf-8\"")
	out.WriteString(ending)
	out.WriteString(">\n")
	if options.css != "" {
		out.WriteString("  <link rel=\"stylesheet\" type=\"text/css\" href=\"")
		attrEscape(out, []byte(options.css))
		out.WriteString("\"")
		out.WriteString(ending)
		out.WriteString(">\n")
	}
	out.WriteString("</head>\n")
	out.WriteString("<body>\n")

	options.tocMarker = out.Len()
}

func (options *Plain) DocumentFooter(out *bytes.Buffer, first bool) {
	if !first {
		return
	}
	// finalize and insert the table of contents
	if options.flags&HTML_TOC != 0 {
		options.TocFinalize()

		// now we have to insert the table of contents into the document
		var temp bytes.Buffer

		// start by making a copy of everything after the document header
		temp.Write(out.Bytes()[options.tocMarker:])

		// now clear the copied material from the main output buffer
		out.Truncate(options.tocMarker)

		// corner case spacing issue
		if options.flags&HTML_COMPLETE_PAGE != 0 {
			out.WriteByte('\n')
		}

		// insert the table of contents
		out.WriteString("<nav>\n")
		out.Write(options.toc.Bytes())
		out.WriteString("</nav>\n")

		// corner case spacing issue
		if options.flags&HTML_COMPLETE_PAGE == 0 && options.flags&HTML_OMIT_CONTENTS == 0 {
			out.WriteByte('\n')
		}

		// write out everything that came after it
		if options.flags&HTML_OMIT_CONTENTS == 0 {
			out.Write(temp.Bytes())
		}
	}

	if options.flags&HTML_COMPLETE_PAGE != 0 {
		out.WriteString("\n</body>\n")
		out.WriteString("</html>\n")
	}
}

func (options *Plain) DocumentMatter(out *bytes.Buffer, matter int) {
	// not used in the Html output
}

func (options *Plain) TocHeaderWithAnchor(text []byte, level int, anchor string) {
	for level > options.currentLevel {
		switch {
		case bytes.HasSuffix(options.toc.Bytes(), []byte("</li>\n")):
			// this sublist can nest underneath a header
			size := options.toc.Len()
			options.toc.Truncate(size - len("</li>\n"))

		case options.currentLevel > 0:
			options.toc.WriteString("<li>")
		}
		if options.toc.Len() > 0 {
			options.toc.WriteByte('\n')
		}
		options.toc.WriteString("<ul>\n")
		options.currentLevel++
	}

	for level < options.currentLevel {
		options.toc.WriteString("</ul>")
		if options.currentLevel > 1 {
			options.toc.WriteString("</li>\n")
		}
		options.currentLevel--
	}

	options.toc.WriteString("<li><a href=\"#")
	if anchor != "" {
		options.toc.WriteString(anchor)
	} else {
		options.toc.WriteString("toc_")
		options.toc.WriteString(strconv.Itoa(options.headerCount))
	}
	options.toc.WriteString("\">")
	options.headerCount++

	options.toc.Write(text)

	options.toc.WriteString("</a></li>\n")
}

func (options *Plain) TocHeader(text []byte, level int) {
	options.TocHeaderWithAnchor(text, level, "")
}

func (options *Plain) TocFinalize() {
	for options.currentLevel > 1 {
		options.toc.WriteString("</ul></li>\n")
		options.currentLevel--
	}

	if options.currentLevel > 0 {
		options.toc.WriteString("</ul>\n")
	}
}

func (options *Plain) SetInlineAttr(i *InlineAttr) {
	options.ial = i
}

func (options *Plain) InlineAttr() *InlineAttr {
	if options.ial == nil {
		return newInlineAttr()
	}
	return options.ial
}

/*
var in string = `Mmark [@mmark] is a markdown processor. It supports the markdown syntax
and has been extended with (syntax) features found in other markdown implementations like
kramdown, PHP markdown extra, [@pandoc], leanpub and even asciidoc.
This allows mmark to be used to write larger, structured documents such
as RFC and I-Ds or even books, while not
deviating too far from markdown`

// we typeset a line, try to put as many words in our 80 column boundery
// and then revisit if we can stretch some spaces so we line out as perfectly
// as possible.

const WIDTH = 30

func main() {
	wo := make([]string, 0, 10) // save word before outputting a line
	var out bytes.Buffer

	all := strings.Replace(in, "\n", " ", -1) // excecpt the first.. ?
	words := strings.Split(all, " ")
	linelen := 0
	for _, word := range words {
		if linelen+len(word) > WIDTH { // line out the current line
			spaces := WIDTH - linelen
			// naive left to right, add spaces
			for i := 0; i < spaces; i++ {
				if i == len(wo)-1 {
					break
				}
				// randomly select word to add space
				wo[rand.Intn(len(wo))] += " "
			}
			for i := 0; i < len(wo); i++ {
				out.WriteString(wo[i])
			}
			out.WriteByte('\n')

			wo = wo[:0]
			linelen = 0
		}
		wo = append(wo, word+" ")
		linelen += len(word) + 1
	}
	// remainder
	for i := 0; i < len(wo); i++ {
		out.WriteString(wo[i])
	}
	out.WriteByte('\n')
	fmt.Printf(out.String())
}
*/
