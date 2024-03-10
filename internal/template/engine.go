// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package template // import "miniflux.app/v2/internal/template"

import (
	"bytes"
	"embed"
	"html/template"
	"io"
	"log/slog"
	"strings"
	textTmpl "text/template"
	"time"

	"miniflux.app/v2/internal/locale"

	"github.com/gorilla/mux"
)

//go:embed templates/common/*.html templates/common/*.rss
var commonTemplateFiles embed.FS

//go:embed templates/views/*.html templates/views/*.rss
var viewTemplateFiles embed.FS

//go:embed templates/standalone/*.html
var standaloneTemplateFiles embed.FS

// Templater is used to provider interoperability between text/template
// and html/template, for the purposes of templating RSS and HTML respectively.
type Templater interface {
	ExecuteTemplate(wr io.Writer, name string, data interface{}) error
	WithFuncs(funcMap map[string]interface{}) Templater
}

type HTMLTemplateWrapper struct {
	T *template.Template
}

func (h HTMLTemplateWrapper) ExecuteTemplate(wr io.Writer, name string, data interface{}) error {
	return h.T.ExecuteTemplate(wr, name, data)
}

func (h HTMLTemplateWrapper) WithFuncs(funcMap map[string]interface{}) Templater {
	h.T = h.T.Funcs(template.FuncMap(funcMap))
	return h
}

type TextTemplateWrapper struct {
	T *textTmpl.Template
}

func (t TextTemplateWrapper) ExecuteTemplate(wr io.Writer, name string, data interface{}) error {
	return t.T.ExecuteTemplate(wr, name, data)
}

func (t TextTemplateWrapper) WithFuncs(funcMap map[string]interface{}) Templater {
	t.T = t.T.Funcs(textTmpl.FuncMap(funcMap))
	return t
}

// Engine handles the templating system.
type Engine struct {
	templates map[string]Templater
	funcMap   *funcMap
}

// NewEngine returns a new template engine.
func NewEngine(router *mux.Router) *Engine {
	return &Engine{
		templates: make(map[string]Templater),
		funcMap:   &funcMap{router},
	}
}

// ParseTemplates parses template files embed into the application.
func (e *Engine) ParseTemplates() error {
	var commonTemplateContents strings.Builder

	dirEntries, err := commonTemplateFiles.ReadDir("templates/common")
	if err != nil {
		return err
	}

	for _, dirEntry := range dirEntries {
		fileData, err := commonTemplateFiles.ReadFile("templates/common/" + dirEntry.Name())
		if err != nil {
			return err
		}
		commonTemplateContents.Write(fileData)
	}

	dirEntries, err = viewTemplateFiles.ReadDir("templates/views")
	if err != nil {
		return err
	}

	for _, dirEntry := range dirEntries {
		templateName := dirEntry.Name()
		fileData, err := viewTemplateFiles.ReadFile("templates/views/" + dirEntry.Name())
		if err != nil {
			return err
		}

		var templateContents strings.Builder
		templateContents.WriteString(commonTemplateContents.String())
		templateContents.Write(fileData)

		slog.Debug("Parsing template",
			slog.String("template_name", templateName),
		)

		if strings.HasSuffix(dirEntry.Name(), ".html") {
			tmpl := template.Must(template.New("main").Funcs(e.funcMap.Map()).Parse(templateContents.String()))
			e.templates[templateName] = HTMLTemplateWrapper{T: tmpl}
		} else {
			tmpl := textTmpl.Must(textTmpl.New("main").Funcs(e.funcMap.Map()).Parse(templateContents.String()))
			e.templates[templateName] = TextTemplateWrapper{T: tmpl}
		}
	}

	dirEntries, err = standaloneTemplateFiles.ReadDir("templates/standalone")
	if err != nil {
		return err
	}

	for _, dirEntry := range dirEntries {
		templateName := dirEntry.Name()
		fileData, err := standaloneTemplateFiles.ReadFile("templates/standalone/" + dirEntry.Name())
		if err != nil {
			return err
		}

		slog.Debug("Parsing template",
			slog.String("template_name", templateName),
		)

		if strings.HasSuffix(dirEntry.Name(), ".html") {
			tmpl := template.Must(template.New("base").Funcs(e.funcMap.Map()).Parse(string(fileData)))
			e.templates[templateName] = HTMLTemplateWrapper{T: tmpl}
		} else {
			tmpl := textTmpl.Must(textTmpl.New("base_rss").Funcs(e.funcMap.Map()).Parse(string(fileData)))
			e.templates[templateName] = TextTemplateWrapper{T: tmpl}
		}
	}

	return nil
}

// Render process a template.
func (e *Engine) Render(name string, data map[string]interface{}) []byte {
	tpl, ok := e.templates[name]
	if !ok {
		panic("This template does not exists: " + name)
	}

	printer := locale.NewPrinter(data["language"].(string))

	// Functions that need to be declared at runtime.
	funcMap := map[string]interface{}{
		"elapsed": func(timezone string, t time.Time) string {
			return elapsedTime(printer, timezone, t)
		},
		"t": func(key interface{}, args ...interface{}) string {
			switch k := key.(type) {
			case string:
				return printer.Printf(k, args...)
			case error:
				return k.Error()
			default:
				return ""
			}
		},
		"plural": func(key string, n int, args ...interface{}) string {
			return printer.Plural(key, n, args...)
		},
	}

	var b bytes.Buffer

	templateName := "base"
	if data["rss"] == true {
		templateName = "base_rss"
		rssTpl := tpl.WithFuncs(funcMap)

		if err := rssTpl.ExecuteTemplate(&b, templateName, data); err != nil {
			panic(err)
		}
	} else {
		htmlTpl := tpl.WithFuncs(funcMap)

		if err := htmlTpl.ExecuteTemplate(&b, templateName, data); err != nil {
			panic(err)
		}
	}

	return b.Bytes()
}
