package main

import "log"

type lexemeType string

const (
	lexemeEof       lexemeType = "EOF"
	lexemeText                 = "TEXT"
	lexemeIf                   = "IF"
	lexemeEndIf                = "ENDIF"
	lexemeEnd                  = "END"
	lexemeExpr                 = "EXPR"
	lexemeInclude              = "INCLUDE"
	lexemeComponent            = "COMPONENT"
	lexemeString               = "STRING"
	lexemeLeftMeta             = "LEFTMETA"
	lexemeRightMeta            = "RIGHTMETA"
)

type lexeme struct {
	typ lexemeType
	val string
}

type lexer struct {
	input   string
	start   int
	pos     int
	width   int
	lexemes chan lexeme
}

type stateFn = (*lexer)

func (l lexeme) String() string {
	return l.val
}

func lex(input string) (*lexer, chan lexeme) {
	l := &lexer{
		input:   input,
		lexemes: make(chan lexeme),
	}

	go l.run()

	return l, l.lexemes
}

func (l *lexer) run() {
	for state := lexText; state != nil; {
		state = state(l)
	}
	close(l.lexemes)
}

func (l *lexer) lexText() stateFn {}

func (l *lexer) emit(typ lexemeType) {
	l.lexemes <- lexeme{typ: typ, val: l.input[l.start:l.pos]}
	l.start = l.pos
}

func main() {
	_, lexemes := lex("hi")

	log.Println(lexemes)
}
