package edge

import (
	"fmt"
	"testing"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
)

func TestLayouts(t *testing.T) {
	m := minify.New()

	m.AddFunc("text/html", html.Minify)

	engine := Default()

	result := engine.Render("home", map[string]any{"title": "hello"})

	fmt.Println(result)

	// a, _ := m.String("text/html", result)

	// b, _ := m.String("text/html", `
	// 	<!DOCTYPE html>
	// 	<html lang="en">
	// 	<head>
	// 		<meta charset="UTF-8">
	// 		<meta name="viewport" content="width=device-width, initial-scale=1.0">
	// 		<meta http-equiv="X-UA-Compatible" content="ie=edge">
	// 	</head>
	// 	<body>
	// 		<h1>hello</h1>
	// 	</body>
	// 	</html>
	// `)

	// assert.Equal(t, a, b, "Should be the same.")
}
