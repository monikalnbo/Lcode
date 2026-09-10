package lexer

import (
	"fmt"
	"strings"

	"lcode/pkg/token"
)

// Lexer 词法分析器结构体
type Lexer struct {
	filename     string
	input        string
	position     int  // 当前字符位置
	readPosition int  // 下一个读取位置
	ch           byte // 当前正在考察的字符
	line         int  // 当前行号 (从1开始)
	column       int  // 当前列号 (从1开始)
	Errors       []string
}

// New 创建一个新的词法分析器实例
func New(filename, input string) *Lexer {
	l := &Lexer{
		filename: filename,
		input:    input,
		line:     1,
		column:   0,
		Errors:   make([]string, 0),
	}
	l.readChar()
	return l
}

// readChar 读取下一个字符并推进位置指针
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII NUL 表示 EOF
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
}

// peekChar 预读下一个字符（不推进指针）
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// currentPos 获取当前 Token 起始位置
func (l *Lexer) currentPos() token.Position {
	return token.Position{
		Filename: l.filename,
		Line:     l.line,
		Column:   l.column,
	}
}

// NextToken 扫描并返回下一个 Token
func (l *Lexer) NextToken() token.Token {
	l.skipWhitespaceAndComments()

	pos := l.currentPos()
	var tok token.Token

	switch l.ch {
	case '+':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.ADD_ASSIGN, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = newToken(token.ADD, l.ch, pos)
		}
	case '-':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.SUB_ASSIGN, Literal: string(ch) + string(l.ch), Pos: pos}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.ARROW, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = newToken(token.SUB, l.ch, pos)
		}
	case '*':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.MUL_ASSIGN, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = newToken(token.MUL, l.ch, pos)
		}
	case '/':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.QUO_ASSIGN, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = newToken(token.QUO, l.ch, pos)
		}
	case '%':
		tok = newToken(token.REM, l.ch, pos)
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.EQL, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = newToken(token.ASSIGN, l.ch, pos)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.NEQ, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = newToken(token.NOT, l.ch, pos)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.LEQ, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = newToken(token.LSS, l.ch, pos)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.GEQ, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = newToken(token.GTR, l.ch, pos)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.LAND, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = newToken(token.AMP, l.ch, pos)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.LOR, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = newToken(token.ILLEGAL, l.ch, pos)
		}
	case ':':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.DEFINE, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = newToken(token.COLON, l.ch, pos)
		}
	case ';':
		tok = newToken(token.SEMICOLON, l.ch, pos)
	case ',':
		tok = newToken(token.COMMA, l.ch, pos)
	case '.':
		tok = newToken(token.DOT, l.ch, pos)
	case '(':
		tok = newToken(token.LPAREN, l.ch, pos)
	case ')':
		tok = newToken(token.RPAREN, l.ch, pos)
	case '{':
		tok = newToken(token.LBRACE, l.ch, pos)
	case '}':
		tok = newToken(token.RBRACE, l.ch, pos)
	case '[':
		tok = newToken(token.LBRACKET, l.ch, pos)
	case ']':
		tok = newToken(token.RBRACKET, l.ch, pos)
	case '"':
		str, err := l.readString()
		if err != nil {
			tok = token.Token{Type: token.ILLEGAL, Literal: str, Pos: pos}
			l.addError(fmt.Sprintf("%s: 未终止的字符串字面量", pos))
		} else {
			tok = token.Token{Type: token.STRING, Literal: str, Pos: pos}
		}
		return tok
	case 0:
		tok = token.Token{Type: token.EOF, Literal: "", Pos: pos}
	default:
		if isLetter(l.ch) {
			ident := l.readIdentifier()
			tokType := token.LookupIdent(ident)
			if tokType == token.TRUE || tokType == token.FALSE {
				return token.Token{Type: token.BOOL, Literal: ident, Pos: pos}
			}
			return token.Token{Type: tokType, Literal: ident, Pos: pos}
		} else if isDigit(l.ch) {
			numLit, numType := l.readNumber()
			return token.Token{Type: numType, Literal: numLit, Pos: pos}
		} else {
			tok = newToken(token.ILLEGAL, l.ch, pos)
			l.addError(fmt.Sprintf("%s: 非法字符 '%c'", pos, l.ch))
		}
	}

	l.readChar()
	return tok
}

func newToken(tokenType token.TokenType, ch byte, pos token.Position) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch), Pos: pos}
}

// skipWhitespaceAndComments 跳过空白与单行/多行注释
func (l *Lexer) skipWhitespaceAndComments() {
	for {
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
			if l.ch == '\n' {
				l.line++
				l.column = 0
			}
			l.readChar()
		}

		// 单行注释 //
		if l.ch == '/' && l.peekChar() == '/' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			continue
		}

		// 多行注释 /* ... */
		if l.ch == '/' && l.peekChar() == '*' {
			l.readChar() // 吃掉 /
			l.readChar() // 吃掉 *
			for {
				if l.ch == 0 {
					l.addError(fmt.Sprintf("%s: 未终止的多行注释", l.currentPos()))
					break
				}
				if l.ch == '\n' {
					l.line++
					l.column = 0
				}
				if l.ch == '*' && l.peekChar() == '/' {
					l.readChar() // 吃掉 *
					l.readChar() // 吃掉 /
					break
				}
				l.readChar()
			}
			continue
		}

		break
	}
}

func (l *Lexer) readIdentifier() string {
	startPos := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[startPos:l.position]
}

func (l *Lexer) readNumber() (string, token.TokenType) {
	startPos := l.position
	isFloat := false

	for isDigit(l.ch) {
		l.readChar()
	}

	// 浮点数识别
	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar() // 吃掉 '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	lit := l.input[startPos:l.position]
	if isFloat {
		return lit, token.FLOAT
	}
	return lit, token.INT
}

func (l *Lexer) readString() (string, error) {
	var sb strings.Builder
	l.readChar() // 跳过前导引号 "

	for l.ch != '"' {
		if l.ch == 0 || l.ch == '\n' {
			return sb.String(), fmt.Errorf("未终止的字符串")
		}
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			default:
				sb.WriteByte('\\')
				sb.WriteByte(l.ch)
			}
		} else {
			sb.WriteByte(l.ch)
		}
		l.readChar()
	}

	l.readChar() // 跳过结束引号 "
	return sb.String(), nil
}

func (l *Lexer) addError(msg string) {
	l.Errors = append(l.Errors, msg)
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
