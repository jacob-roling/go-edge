package edge

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-gonic/gin/render"
	"github.com/maja42/goval"
)

type Template func(any) (string, error)

type Config struct {
	BaseDirectory string
	Functions     map[string]goval.ExpressionFunction
}

type Edge struct {
	BaseDirectory string
	Cache         map[string]Template
	Eval          *goval.Evaluator
	Functions     map[string]goval.ExpressionFunction
}

func New(config Config) Edge {
	return Edge{
		BaseDirectory: config.BaseDirectory,
		Cache:         make(map[string]Template),
		Eval:          goval.NewEvaluator(),
		Functions:     config.Functions,
	}
}

func Default() Edge {
	return Edge{
		BaseDirectory: "views",
		Cache:         make(map[string]Template),
		Eval:          goval.NewEvaluator(),
		Functions:     nil,
	}
}

func extractTagContents(templateString string) []rune {
	contents := make([]rune, 0)

	for _, char := range templateString {
		switch char {
		case 39:
			return contents
		}
		contents = append(contents, char)
	}

	return contents
}

func extractTag(templateString string) ([]rune, []rune) {
	tag := make([]rune, 0)

	for index, char := range templateString {
		switch char {
		case 40:
			return tag, extractTagContents(templateString[index+2:])
		}
		if index > 2 {
			switch string(templateString[index-3 : index]) {
			case "end":
				return tag, tag
			}
		}
		tag = append(tag, char)
	}

	return tag, tag
}

func extractSection(templateString string) []rune {
	section := make([]rune, 0)

	for index, char := range templateString {
		switch char {
		case 64:
			tag, _ := extractTag(templateString[index+1:])
			switch string(tag) {
			case "end":
				return section
			}
		}
		section = append(section, char)
	}

	return section
}

func extractExpression(templateString string) ([]rune, bool) {
	expression := make([]rune, 0)

	if templateString[0] != 123 {
		return expression, false
	}

	for index, char := range templateString {
		if index > 1 {
			switch string(templateString[index-2 : index]) {
			case "}}":
				return expression[1 : len(expression)-3], true
			}
		}
		expression = append(expression, char)
	}

	return expression, false
}

func (edge *Edge) Compile(templateString string) Template {

	layout := make([]rune, 0)
	history := make([]rune, 0)
	sections := make(map[string]string)
	template := func(data any) (string, error) {
		return "", nil
	}
	ignoreUntil := 0

	for index, char := range templateString {
		if index < ignoreUntil {
			continue
		}
		history = append(history, char)
		switch char {
		case 64:
			tag, tagContents := extractTag(templateString[index+1:])

			switch string(tag) {
			case "layout":
				layout = tagContents
			case "section":
				sectionContents := extractSection(templateString[index+len(tag)+len(tagContents)+5:])
				sections[string(tagContents)] = strings.TrimSpace(string(sectionContents))
			}
		case 123:
			expression, ok := extractExpression(templateString[index+1:])
			if ok {
				oldTemplate := template
				frozenHistory := string(history[:len(history)-1])
				template = func(data any) (string, error) {
					result, err := edge.Eval.Evaluate(string(expression), data.(map[string]any), edge.Functions)

					if err != nil {
						return "", err
					}

					oldString, err := oldTemplate(data)

					if err != nil {
						return "", err
					}

					return oldString + frozenHistory + fmt.Sprintf("%v", result), nil
				}
				ignoreUntil = index + len(expression) + 5
				history = nil
			}
		}
	}

	if len(layout) > 0 {
		history = nil
		newTemplateString := edge.Render(string(layout), nil)
		ignoreUntil := 0
		for index, char := range newTemplateString {
			if index < ignoreUntil {
				continue
			}
			switch char {
			case 64:
				tag, tagContents := extractTag(newTemplateString[index+1:])
				switch string(tag) {
				case "!section":
					contents, ok := sections[string(tagContents)]
					if ok {
						history = append(history, []rune(contents)...)
					}
					ignoreUntil = index + len(tag) + len(tagContents) + 5
				default:
					history = append(history, char)
				}
			default:
				history = append(history, char)
			}
		}
		return edge.Compile(string(history))
	} else {
		oldTemplate := template
		frozenHistory := string(history)
		template = func(data any) (string, error) {
			oldString, err := oldTemplate(data)
			if err != nil {
				return "", err
			}
			return oldString + frozenHistory, nil
		}
	}

	return template
}

func (template Template) Exec(data any) (string, error) {
	return template(data)
}

func (edge *Edge) Render(templateName string, data any) string {
	template, ok := edge.Cache[templateName]

	if ok {
		result, err := template(data)
		if err != nil {
			panic(err)
		}
		return result
	}

	bytes, err := os.ReadFile(path.Join(edge.BaseDirectory, templateName+".edge"))

	if err != nil {
		panic(err)
	}

	edge.Cache[templateName] = edge.Compile(string(bytes))

	result, err := edge.Cache[templateName](data)

	if err != nil {
		panic(err)
	}

	return result
}

type EdgeGin struct {
	TemplateName string
	Context      any
	Edge         *Edge
}

func (edgeGin EdgeGin) Instance(templateName string, data any) render.Render {
	edgeGin.TemplateName = templateName
	edgeGin.Context = data
	return edgeGin
}

func (edgeGin EdgeGin) Render(w http.ResponseWriter) error {
	output := edgeGin.Edge.Render(edgeGin.TemplateName, edgeGin.Context)

	w.Write([]byte(output))

	return nil
}

func (edgeGin EdgeGin) WriteContentType(w http.ResponseWriter) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{"text/html; charset=utf-8"}
	}
}
