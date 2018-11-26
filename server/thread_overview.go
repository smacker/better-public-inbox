package server

const threadOverviewTpl = `
{{define "threadOverview"}}
<strong>Thread overview</strong>: {{ . | len }}+ messages / expand  <a href="#b">top</a>
-- links below jump to the message on this page --
{{range .}}{{ .Date.Format "2006-01-02 15:04" }} {{ repeat  "  " .Level }}<a id="r{{ .ID | idshort }}" href="#m{{ .ID | idshort }}">{{ .Title }}</a>
{{end}}
{{end}}`
