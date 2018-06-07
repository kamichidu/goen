package generator

import (
	"path"
	"text/template"
)

var (
	debug = ""

	templates = template.New("goen")
)

//go:generate esc -o bindata.go -pkg generator -private templates/
func init() {
	// it panics on useLocal is true, since cwd is mismatched
	useLocal := false
	for abspath, file := range _escData {
		if file.IsDir() {
			continue
		} else if path.Ext(abspath) != ".tgo" {
			continue
		}
		templates = template.Must(
			templates.New(path.Base(abspath)).
				Parse(_escFSMustString(useLocal, abspath)))
	}
}
