package edge

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/maja42/goval"
)

type Template func(any) string

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

func newEdge(config Config) Edge {
	return Edge{
		BaseDirectory: config.BaseDirectory,
		Cache:         make(map[string]Template),
		Eval:          goval.NewEvaluator(),
		Functions:     config.Functions,
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
	template := func(data any) string {
		return ""
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
				template = func(data any) string {
					result, err := edge.Eval.Evaluate(string(expression), data.(map[string]any), edge.Functions)
					if err != nil {
						panic(err)
					}
					return oldTemplate(data) + frozenHistory + fmt.Sprintf("%v", result)
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
				}
			default:
				history = append(history, char)
			}
		}
		return edge.Compile(string(history))
	} else {
		oldTemplate := template
		frozenHistory := string(history)
		template = func(data any) string {
			return oldTemplate(data) + frozenHistory
		}
	}

	return template
}

func (edge *Edge) Render(templateName string, data any) string {
	template, ok := edge.Cache[templateName]

	if ok {
		return template(data)
	}

	bytes, err := os.ReadFile(path.Join(edge.BaseDirectory, templateName+".edge"))

	if err != nil {
		panic(err)
	}

	edge.Cache[templateName] = edge.Compile(string(bytes))

	return edge.Cache[templateName](data)
}