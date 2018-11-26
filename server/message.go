package server

import (
	"html/template"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/smacker/better-public-inbox"
)

func (s *HTTPServer) msgHandler(w http.ResponseWriter, r *http.Request) error {
	t, err := template.Must(baseT.Clone()).Parse(msgTpl)
	if err != nil {
		return err
	}

	id := chi.URLParam(r, "id")

	m, err := s.ts.Get(id)
	if err != nil {
		return err
	}

	return t.Execute(w, struct {
		Msg *bpi.Message
	}{Msg: m})
}

const msgTpl = `
{{define "title"}}{{ .Msg.Title }}{{end}}
{{define "content"}}
<pre id="b">
From: {{ .Msg.Author.Name }} <{{ .Msg.Author.Address }}>
To: {{ .Msg.To }}
Subject: <a href="#r">{{ .Msg.Title }}</a>
Date: {{ .Msg.Date }}
Message-ID: <{{ .Msg.ID }}> (<a href="raw">raw</a>)

{{ .Msg.Description }}
{{ .Msg.Patch | htmlPatch }}
</pre>
<hr>
<pre>
<a href="" rel="next">next</a>             <a href="#r">reply</a> <a href="../">index</a>
</pre>
<hr>
{{template "replyInstructions"}}
{{end}}`
