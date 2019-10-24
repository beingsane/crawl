// Extract links from HTML/CSS content.

package analysis

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"git.autistici.org/ale/crawl"
)

var (
	urlcssRx = regexp.MustCompile(`url\(['"]?(.*?)['"]?\)`)

	linkMatches = []struct {
		tag     string
		attr    string
		linkTag int
	}{
		{"a", "href", crawl.TagPrimary},
		{"link", "href", crawl.TagRelated},
		{"img", "src", crawl.TagRelated},
		{"script", "src", crawl.TagRelated},
		{"iframe", "src", crawl.TagRelated},
		{"source", "src", crawl.TagRelated},
		{"object", "data", crawl.TagRelated},
	}
)

// The unparsed version of an Outlink.
type rawOutlink struct {
	URL string
	Tag int
}

// GetLinks returns all the links found in a document. Currently only
// parses HTML pages and CSS stylesheets.
func GetLinks(resp *http.Response) ([]crawl.Outlink, error) {
	// Parse outbound links relative to the request URI, and
	// return unique results.
	var result []crawl.Outlink
	links := make(map[string]crawl.Outlink)
	for _, l := range extractLinks(resp) {
		// Skip data: URLs altogether.
		if strings.HasPrefix(l.URL, "data:") {
			continue
		}
		if linkurl, err := resp.Request.URL.Parse(l.URL); err == nil {
			links[linkurl.String()] = crawl.Outlink{
				URL: linkurl,
				Tag: l.Tag,
			}
		}
	}
	for _, l := range links {
		result = append(result, l)
	}
	return result, nil
}

func extractLinks(resp *http.Response) []rawOutlink {
	ctype := resp.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(ctype, "text/html"):
		return extractLinksFromHTML(resp.Body, nil)
	case strings.HasPrefix(ctype, "text/css"):
		return extractLinksFromCSS(resp.Body, nil)
	default:
		return nil
	}
}

func extractLinksFromHTML(r io.Reader, outlinks []rawOutlink) []rawOutlink {
	// Use goquery to extract links from the parsed HTML contents
	// (query patterns are described in the linkMatches table).
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil
	}
	for _, lm := range linkMatches {
		doc.Find(fmt.Sprintf("%s[%s]", lm.tag, lm.attr)).Each(func(i int, s *goquery.Selection) {
			val, _ := s.Attr(lm.attr)
			outlinks = append(outlinks, rawOutlink{URL: val, Tag: lm.linkTag})
		})
	}

	// Find the inline <style> sections and parse them separately as CSS.
	doc.Find("style").Each(func(i int, s *goquery.Selection) {
		outlinks = extractLinksFromCSS(strings.NewReader(s.Text()), outlinks)
	})

	return outlinks
}

func extractLinksFromCSS(r io.Reader, outlinks []rawOutlink) []rawOutlink {
	// Use a simple (and actually quite bad) regular expression to
	// extract "url()" and "@import" links from CSS.
	if data, err := ioutil.ReadAll(r); err == nil {
		for _, val := range urlcssRx.FindAllStringSubmatch(string(data), -1) {
			outlinks = append(outlinks, rawOutlink{URL: val[1], Tag: crawl.TagRelated})
		}
	}
	return outlinks
}
