package parser

import (
	"fmt"
)

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // current line number
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input, line: 1}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition += 1
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: EQ, Literal: literal, Line: l.line}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: FAT_ARROW, Literal: literal, Line: l.line}
		} else {
			tok = l.newToken(ASSIGN, l.ch)
		}
	case ';':
		tok = l.newToken(SEMICOLON, l.ch)
	case '\n':
		tok = l.newToken(NEWLINE, l.ch)
		l.line++ // Increment line on newline token
	case '(':
		tok = l.newToken(LPAREN, l.ch)
	case ')':
		tok = l.newToken(RPAREN, l.ch)
	case ',':
		tok = l.newToken(COMMA, l.ch)
	case ':':
		if l.peekChar() == ':' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: DOUBLE_COLON, Literal: literal, Line: l.line}
		} else {
			tok = l.newToken(COLON, l.ch)
		}
	case '?':
		if l.peekChar() == '?' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: NULL_COALESCE, Literal: literal, Line: l.line}
		} else {
			tok = l.newToken(QUESTION, l.ch)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: NOT_EQ, Literal: literal, Line: l.line}
		} else {
			tok = l.newToken(BANG, l.ch)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: LTE, Literal: literal, Line: l.line}
		} else if l.peekChar() == '<' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: SHIFT_LEFT, Literal: literal, Line: l.line}
		} else {
			tok = l.newToken(LT, l.ch)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: GTE, Literal: literal, Line: l.line}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: SHIFT_RIGHT, Literal: literal, Line: l.line}
		} else {
			tok = l.newToken(GT, l.ch)
		}
	case '+':
		if l.peekChar() == '+' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: INCREMENT, Literal: literal, Line: l.line}
		} else {
			tok = l.newToken(PLUS, l.ch)
		}
	case '-':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: ARROW, Literal: literal, Line: l.line}
		} else {
			tok = l.newToken(MINUS, l.ch)
		}
	case '*':
		tok = l.newToken(ASTERISK, l.ch)
	case '/':
		if l.peekChar() == '/' {
			l.skipComment()
			return l.NextToken()
		}
		tok = l.newToken(SLASH, l.ch)
	case '%':
		tok = l.newToken(PERCENT, l.ch)
	case '{':
		tok = l.newToken(LBRACE, l.ch)
	case '}':
		tok = l.newToken(RBRACE, l.ch)
	case '[':
		tok = l.newToken(LBRACKET, l.ch)
	case ']':
		tok = l.newToken(RBRACKET, l.ch)
	case '.':
		tok = l.newToken(DOT, l.ch)
	case '$':
		tok = Token{Type: VAR, Literal: "$", Line: l.line}
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString('"')
		tok.Line = l.line
	case '\'':
		tok.Type = STRING
		tok.Literal = l.readString('\'')
		tok.Line = l.line
	case 0:
		tok.Literal = ""
		tok.Type = EOF
		tok.Line = l.line
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: AND, Literal: literal, Line: l.line}
		} else {
			tok = l.newToken(ILLEGAL, l.ch)
		}
	case '|':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: PIPE, Literal: literal, Line: l.line}
		} else if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: OR, Literal: literal, Line: l.line}
		} else {
			tok = l.newToken(ILLEGAL, l.ch)
		}
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			tok.Line = l.line
			return tok
		} else if isDigit(l.ch) {
			tok.Literal = l.readNumber()
			if l.ch == '.' && isDigit(l.peekChar()) {
				l.readChar()
				tok.Literal += "." + l.readNumber()
				tok.Type = FLOAT
			} else {
				tok.Type = INT
			}
			tok.Line = l.line
			return tok
		} else {
			fmt.Printf("ILLEGAL CHAR: %q (%d)\n", l.ch, l.ch)
			tok = l.newToken(ILLEGAL, l.ch)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	l.skipWhitespace()
}

func (l *Lexer) newToken(tokenType TokenType, ch byte) Token {
	return Token{Type: tokenType, Literal: string(ch), Line: l.line}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch == '@'
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func (l *Lexer) readString(delimiter byte) string {
	var out []byte
	for {
		l.readChar()
		if l.ch == delimiter || l.ch == 0 {
			break
		}

		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				out = append(out, '\n')
			case 't':
				out = append(out, '\t')
			case 'r':
				out = append(out, '\r')
			case '"':
				out = append(out, '"')
			case '\'':
				out = append(out, '\'')
			case '\\':
				out = append(out, '\\')
			default:
				out = append(out, '\\')
				out = append(out, l.ch)
			}
		} else {
			out = append(out, l.ch)
		}
	}
	return string(out)
}
