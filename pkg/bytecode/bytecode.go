package bytecode

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/jossecurity/joss/pkg/parser"
)

const MaxProgramSize = 32 << 20

var (
	magic        = []byte{'J', 'O', 'S', 'S', 'B', 'C', '2', 0}
	registerOnce sync.Once
)

// Encode compiles a parsed Joss program into the stable JP v2 bytecode payload.
// It stores the AST, not the original source text.
func Encode(program *parser.Program) ([]byte, error) {
	if program == nil {
		return nil, fmt.Errorf("bytecode: programa nil")
	}
	registerAST()
	var body bytes.Buffer
	if err := gob.NewEncoder(&body).Encode(program); err != nil {
		return nil, fmt.Errorf("bytecode: no se pudo codificar: %w", err)
	}
	if body.Len() > MaxProgramSize {
		return nil, fmt.Errorf("bytecode: el programa excede %d MiB", MaxProgramSize>>20)
	}
	result := make([]byte, 0, len(magic)+body.Len())
	result = append(result, magic...)
	result = append(result, body.Bytes()...)
	return result, nil
}

// Decode restores a precompiled Joss program from a JP v2 payload.
func Decode(data []byte) (*parser.Program, error) {
	if len(data) < len(magic) || !bytes.Equal(data[:len(magic)], magic) {
		return nil, fmt.Errorf("bytecode: cabecera JP v2 invalida")
	}
	if len(data)-len(magic) > MaxProgramSize {
		return nil, fmt.Errorf("bytecode: el programa excede %d MiB", MaxProgramSize>>20)
	}
	registerAST()
	var program parser.Program
	if err := gob.NewDecoder(bytes.NewReader(data[len(magic):])).Decode(&program); err != nil {
		return nil, fmt.Errorf("bytecode: payload invalido: %w", err)
	}
	return &program, nil
}

func IsBytecode(data []byte) bool {
	return len(data) >= len(magic) && bytes.Equal(data[:len(magic)], magic)
}

func registerAST() {
	registerOnce.Do(func() {
		// Statements stored behind parser.Statement interfaces.
		gob.Register(&parser.LetStatement{})
		gob.Register(&parser.MultiLetStatement{})
		gob.Register(&parser.ExpressionStatement{})
		gob.Register(&parser.ClassStatement{})
		gob.Register(&parser.BlockStatement{})
		gob.Register(&parser.EchoStatement{})
		gob.Register(&parser.InitStatement{})
		gob.Register(&parser.ForeachStatement{})
		gob.Register(&parser.ImportStatement{})
		gob.Register(&parser.MethodStatement{})
		gob.Register(&parser.WhileStatement{})
		gob.Register(&parser.DoWhileStatement{})
		gob.Register(&parser.TryCatchStatement{})
		gob.Register(&parser.ThrowStatement{})
		gob.Register(&parser.ReturnStatement{})
		gob.Register(&parser.BreakStatement{})
		gob.Register(&parser.ContinueStatement{})

		// Expressions stored behind parser.Expression interfaces.
		gob.Register(&parser.Identifier{})
		gob.Register(&parser.StringLiteral{})
		gob.Register(&parser.CallExpression{})
		gob.Register(&parser.TernaryExpression{})
		gob.Register(&parser.InfixExpression{})
		gob.Register(&parser.PrefixExpression{})
		gob.Register(&parser.PostfixExpression{})
		gob.Register(&parser.Boolean{})
		gob.Register(&parser.IntegerLiteral{})
		gob.Register(&parser.FloatLiteral{})
		gob.Register(&parser.ArrayLiteral{})
		gob.Register(&parser.MapLiteral{})
		gob.Register(&parser.IndexExpression{})
		gob.Register(&parser.FunctionLiteral{})
		gob.Register(&parser.NewExpression{})
		gob.Register(&parser.MemberExpression{})
		gob.Register(&parser.AssignExpression{})
		gob.Register(&parser.IssetExpression{})
		gob.Register(&parser.EmptyExpression{})
		gob.Register(&parser.BlockExpression{})
		gob.Register(&parser.MatchExpression{})
	})
}
