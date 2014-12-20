// Extract links from HTML/CSS content.

package analysis

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	urlcssRx = regexp.MustCompile(`background.*:.*url\(["']?([^'"\)]+)["']?\)`)

	linkMatches = []struct {
		tag  string
		attr string
	}{
		{"a", "href"},
		{"link", "href"},
		{"img", "src"},
		{"script", "src"},
	}
)

func GetLinks(resp *http.Response) ([]*url.URL, error) {
	var outlinks []string

	ctype := resp.Header.Get("Content-Type")
	if strings.HasPrefix(ctype, "text/html") {
		doc, err := goquery.NewDocumentFromResponse(resp)
		if err != nil {
			return nil, err
		}

		for _, lm := range linkMatches {
			doc.Find(fmt.Sprintf("%s[%s]", lm.tag, lm.attr)).Each(func(i int, s *goquery.Selection) {
				val, _ := s.Attr(lm.attr)
				outlinks = append(outlinks, val)
			})
		}
	} else if strings.HasPrefix(ctype, "text/css") {
		if data, err := ioutil.ReadAll(resp.Body); err == nil {
			for _, val := range urlcssRx.FindAllStringSubmatch(string(data), -1) {
				outlinks = append(outlinks, val[1])
			}
		}
	}

	// Uniquify and parse outbound links.
	var result []*url.URL
	links := make(map[string]*url.URL)
	for _, val := range outlinks {
		if linkurl, err := resp.Request.URL.Parse(val); err == nil {
			links[linkurl.String()] = linkurl
		}
	}
	for _, link := range links {
		result = append(result, link)
	}

	return result, nil
}
