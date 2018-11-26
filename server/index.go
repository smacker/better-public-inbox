package server

import (
	"html/template"
	"net/http"

	"github.com/smacker/better-public-inbox"
)

func (s *HTTPServer) indexHandler(w http.ResponseWriter, r *http.Request) error {
	t, err := template.Must(baseT.Clone()).Parse(indexTpl)
	if err != nil {
		return err
	}

	list, err := s.ts.List()
	if err != nil {
		return err
	}

	items := make([]*indexTplItem, len(list))
	for i, m := range list {
		count, err := s.ts.ThreadCount(m.ID)
		if err != nil {
			return err
		}

		items[i] = &indexTplItem{
			MessageHeader: m,
			ThreadCount:   count,
		}
	}

	return t.Execute(w, indexTplData{
		Items: items,
	})
}

type indexTplItem struct {
	*bpi.MessageHeader
	ThreadCount int
}

type indexTplData struct {
	Items []*indexTplItem
}

const indexTpl = `
{{define "title"}}Test{{end}}
{{define "content"}}
<pre>
{{range .Items}}
<a href="{{ .ID }}/T/"><strong>{{ .Title }}</strong></a>
{{ .Date.UTC.Format "2006-01-02 15:04:05 UTC" }} ({{ .ThreadCount }}+ messages) - <a href="#">mbox.gz</a>
{{else}}
No messages
{{end}}
</pre>
{{end}}`
