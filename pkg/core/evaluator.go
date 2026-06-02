package core

import (
	"github.com/jossecurity/joss/pkg/parser"
)

func (r *Runtime) evaluateExpression(exp parser.Expression) interface{} {
	switch e := exp.(type) {
	case *parser.StringLiteral:
		return e.Value
	case *parser.IntegerLiteral:
		return e.Value
	case *parser.FloatLiteral:
		return e.Value
	case *parser.Boolean:
		return e.Value
	case *parser.CallExpression:
		return r.executeCall(e)
	case *parser.Identifier:
		if val, ok := r.Variables[e.Value]; ok {
			return val
		}
		return nil
	case *parser.TernaryExpression:
		return r.evaluateTernary(e)
	case *parser.InfixExpression:
		return r.evaluateInfix(e)
	case *parser.ArrayLiteral:
		return r.evaluateArray(e)
	case *parser.MapLiteral:
		return r.evaluateMap(e)
	case *parser.IndexExpression:
		return r.evaluateIndex(e)
	case *parser.NewExpression:
		return r.evaluateNew(e)
	case *parser.MemberExpression:
		return r.evaluateMember(e)
	case *parser.AssignExpression:
		return r.evaluateAssign(e)
	case *parser.IssetExpression:
		return r.evaluateIsset(e)
	case *parser.EmptyExpression:
		return r.evaluateEmpty(e)
	case *parser.BlockExpression:
		// Return the block itself (or a closure wrapper if we had one)
		// For now, just return the BlockStatement so Task can execute it.
		return e.Block
	case *parser.FunctionLiteral:
		return e
	case *parser.PrefixExpression:
		return r.evaluatePrefix(e)
	case *parser.PostfixExpression:
		return r.evaluatePostfix(e)
	case *parser.MatchExpression:
		return r.evaluateMatch(e)
	}
	return nil
}
