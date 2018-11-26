package server

import (
	"html/template"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/smacker/better-public-inbox"
)

func (s *HTTPServer) threadHandler(w http.ResponseWriter, r *http.Request) error {
	t, err := template.Must(baseT.Clone()).Parse(threadTpl)
	if err != nil {
		return err
	}

	id := chi.URLParam(r, "id")

	root, err := s.ts.Thread(id)
	if err != nil {
		return err
	}

	list := root.List()

	return t.Execute(w, struct {
		Root        *bpi.Message
		Items       []*bpi.TreeMessage
		ThreadCount int
	}{
		Root:        list[0].Message,
		Items:       list,
		ThreadCount: len(list),
	})
}

const threadTpl = `
{{define "title"}}{{ .Root.Title }}{{end}}
{{define "content"}}{{range $i, $e := .Items}}
<pre {{if not $i}}id="b"{{end}}>
<a id="m{{ .ID | idshort }}" href="e{{ .ID | idshort }}">*</a> <strong>{{ .Title }}</strong>
From: {{ .Author.Name }} @ {{ .Date.Format "2006-01-02 15:04:05 UTC" }} (<a href="">permalink</a> / <a href="">raw</a>)
  To: {{ .To }}; <strong>+Cc:</strong> {{ .Cc }}
</pre>
{{range .Body }}
{{renderBlock .Body .Type }}
{{end}}
<pre>
<a id="e{{ .ID | idshort }}" href="m{{ .ID | idshort }}">^</a> <a href="../../{{ .ID }}/">permalink</a> <a href="../../{{ .ID }}/raw">raw</a>  <a href="../../{{ .ID }}/#R">reply</a>	<a href="#r{{ .ID | idshort }}">{{ $.ThreadCount }}+ messages in thread</a>
</pre>
<hr>{{end}}
<pre>
end of thread, back to <a href="../..">index</a>

{{template "threadOverview" .Items}}
</pre>
{{end}}
`
