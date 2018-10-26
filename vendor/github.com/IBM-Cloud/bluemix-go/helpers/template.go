package helpers

import (
	"bytes"
	"html/template"
)

func Tprintf(tmpl string, param map[string]interface{}) string {
	t := template.Must(template.New("Tprintf").Parse(tmpl))
	buf := &bytes.Buffer{}
	if err := t.Execute(buf, param); err != nil {
		return tmpl
	}
	return buf.String()
}
