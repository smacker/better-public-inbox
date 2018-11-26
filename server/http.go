package server

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"html/template"
	"net/http"
	"strings"

	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/smacker/better-public-inbox"
)

type HTTPServer struct {
	ts  bpi.Store
	mux http.Handler
}

func NewHTTPServer(ts bpi.Store) *HTTPServer {
	r := chi.NewRouter()
	s := &HTTPServer{
		ts:  ts,
		mux: r,
	}

	r.Use(middleware.StripSlashes)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", render(s.indexHandler))
	r.Get("/{id}", render(s.msgHandler))
	r.Get("/{id}/T", render(s.threadHandler))
	r.Get("/favicon.ico", http.NotFound)

	return s
}

func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func render(handler func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

var funcs = template.FuncMap{
	"idshort":     idshort,
	"htmlDiff":    htmlDiff,
	"repeat":      strings.Repeat,
	"renderBlock": renderBlock,
}

func idshort(id string) string {
	h := sha1.New()
	h.Write([]byte(id))
	return hex.EncodeToString(h.Sum(nil))
}

func htmlDiff(diff string) interface{} {
	html, err := formatDiff(diff)
	if err != nil {
		return diff
	}
	return template.HTML(html)
}

func formatDiff(diff string) (string, error) {
	if diff == "" {
		return "", nil
	}

	lexer := lexers.Get("diff")
	style := styles.Get("pygments")
	formatter := html.New()
	iterator, err := lexer.Tokenise(nil, diff)
	if err != nil {
		return "", nil
	}

	buf := bytes.NewBuffer(nil)
	err = formatter.Format(buf, style, iterator)
	if err != nil {
		return "", nil
	}

	return buf.String(), nil
}

func renderBlock(body, t string) interface{} {
	switch t {
	case "quotes":
		return template.HTML("<pre class='quotes'>" + template.HTMLEscapeString(body) + "</pre>")
	case "patch":
		diffs, err := bpi.ParseDiff(body)
		if err != nil {
			return body
		}

		var result []string
		for _, diff := range diffs {
			html, err := formatDiff(diff)
			if err != nil {
				html = "<pre>" + template.HTMLEscapeString(diff) + "</pre>"
			}
			result = append(result, html)
		}

		return template.HTML(strings.Join(result, ""))
	default:
		return template.HTML("<pre>" + template.HTMLEscapeString(body) + "</pre>")
	}
}

const baseTpl = `{{define "base"}}
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{block "title" .}}Test{{end}}</title>
		<style>pre{white-space:pre-wrap}.quotes{color:#999};</style>
	</head>
	<body>
	{{template "content" .}}
	</body>
</html>
{{end}}`

var baseT = template.Must(template.Must(template.Must(template.New("base").
	Funcs(funcs).
	Parse(baseTpl)).
	Parse(replyInstructionsTpl)).
	Parse(threadOverviewTpl))

const replyInstructionsTpl = `
{{define "replyInstructions"}}
<pre id="R"><string>Reply instructions:</strong>

You may reply publically to <a href="#t">this message</a> via plain-text email
using any one of the following methods:

* Save the following mbox file, import it into your mail client,
  and reply-to-all from there: <a href="raw">mbox</a>

  Avoid top-posting and favor interleaved quoting:
  <a href="https://en.wikipedia.org/wiki/Posting_style#Interleaved_style">https://en.wikipedia.org/wiki/Posting_style#Interleaved_style</a>

  List information: <a href="https://public-inbox.org/README">https://public-inbox.org/README</a>

* Reply using the <b>--to</b>, <b>--cc</b>, and <b>--in-reply-to</b>
  switches of git-send-email(1):

  git send-email \
    --in-reply-to='r6a3cRnCmsfOm9mlbjvsQu4T2g2041vFpSGczjpOTuoxUYjtijyZcLZyz4f5Bc8dt45ePRSFsHWVL9RlKId6q9GcnFAlQ_Cd-x0ZBk4s27E=@protonmail.com' \
    --to=yscheffer@protonmail.com \
    --cc=meta@public-inbox.org \
    /path/to/YOUR_REPLY

  <a href="https://kernel.org/pub/software/scm/git/docs/git-send-email.html">https://kernel.org/pub/software/scm/git/docs/git-send-email.html</a>

* If your mail client supports setting the <b>In-Reply-To</b> header
  via mailto: links, try the <a href="">mailto: link</a>
</pre>
{{end}}`
