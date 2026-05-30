package http

import (
	"embed"
	"fmt"
	"html/template"
	"sync"

	app "github.com/oriolf/simple-app"
)

var (
	TEMPLATES      = make(map[string]*template.Template)
	TEMPLATES_LOCK sync.Mutex

	templateFuncs = map[string]any{
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("invalid dict call")
			}
			dict := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}
)

func InitTemplates(templateFiles embed.FS, templateFuncs ...map[string]any) app.Option {
	return func() error {
		funcs := map[string]any{}
		if templateFuncs != nil {
			funcs = templateFuncs[0]
		}
		if err := initTemplates(templateFiles, funcs); err != nil {
			return fmt.Errorf("could not initialize templates: %w", err)
		}
		return nil
	}
}

func getTemplate(filename string) *template.Template {
	TEMPLATES_LOCK.Lock()
	defer TEMPLATES_LOCK.Unlock()
	return TEMPLATES[filename]
}

func initTemplates(templateFiles embed.FS, userTemplateFuncs map[string]any) error {
	TEMPLATES_LOCK.Lock()
	defer TEMPLATES_LOCK.Unlock()

	files, err := templateFiles.ReadDir("templates")
	if err != nil {
		return fmt.Errorf("could not read dir: %w", err)
	}

	funcs := app.MergeMaps(templateFuncs, userTemplateFuncs)
	for _, f := range files {
		name := f.Name()
		tmpl, err := template.New(name).Funcs(funcs).ParseFS(templateFiles, "templates/layout.html", "templates/"+name)
		if err != nil {
			return fmt.Errorf("could not parse template %s: %w", name, err)
		}
		TEMPLATES[name] = tmpl
	}

	return nil
}

func Template(filename string) func(Request) Response {
	return func(r Request) Response {
		return r.TemplateResponse(filename, nil, nil)
	}
}

func TemplateList[C any, T app.Lister[T, C]](filename string, seed T) func(Request) Response {
	return func(r Request) Response {
		paginator := app.NewPaginator(r.MustParameters())
		items, total, err := seed.List(paginator, getFilterCriteria(r, seed))
		if err != nil {
			return r.TemplateError(err)
		}
		paginator.SetTotal(total)
		return r.TemplateResponse(filename, map[string]any{"items": items, "total": total, "paginator": paginator}, nil)
	}
}
