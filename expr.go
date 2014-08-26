// Package expr provides runtime evaluation of Go-like expressions against
// named values.
//
// eg.
//
// 		expr := MustCompile("a + 1 > 2")
// 		expr.Bool(V{"a": 0}) == false
// 		expr.Bool(V{"a": 1}) == false
// 		expr.Bool(V{"a": 2}) == true
//
package expr

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
)

// V is a (possibly nested) map of name:value pairs that are evaluated against
// an Expression.
type V map[string]interface{}

// Expression is a expression that is compiled and ready for evaluation.
type Expression struct {
	ast   *ast.Expr
	Expr  string
	Terms []string // Collected terms from the expression. May contain duplicates.
}

func MustCompile(expr string) *Expression {
	m, err := Compile(expr)
	if err != nil {
		panic(fmt.Errorf("%s: %s", expr, err.Error()))
	}
	return m
}

// Compile creates a new Expression for evaluating an expression against a
// value. An empty expression always evaluates to true.
//
// An expression is any syntactically valid Go expression (excluding the
// subscript operator []). Nested Values can be traversed with A.B.C.
func Compile(expr string) (*Expression, error) {
	e := &Expression{
		Expr: expr,
	}
	err := e.compile()
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Expression) String() string {
	return e.Expr
}

func (e *Expression) compile() error {
	if e.Expr == "" {
		return nil
	}
	ast, err := parser.ParseExpr(e.Expr)
	if err != nil {
		return err
	}
	e.ast = &ast
	index(ast, &e.Terms)
	return nil
}

// Bool evaluates the expression against a value and returns its "truthiness".
// The empty expression evaluates to true. Any errors will evaluate to false.
func (e *Expression) Bool(value V) bool {
	if e.Expr == "" {
		return true
	}
	v, err := e.Eval(value)
	if err != nil {
		return false
	}
	return boolCast(v)
}

// Eval evaluates an expression against a value and returns the final result.
//
// Type evaluation is much less strict than Go. In particular, signed
// integers, unsigned integers and float values are converted to the largest
// super type. So, int8 -> int64, float32 -> float64, and so on. Additionally,
// nil values are treated as false.
func (e *Expression) Eval(value V) (v interface{}, err error) {
	if e.Expr == "" {
		return "", nil
	}
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()
	v = eval(value, *e.ast)
	return
}

// Normalize all int values to int64, all unsigned int values to uint64 and
// all float values to float64.
func normalize(v interface{}) interface{} {
	switch rv := v.(type) {
	case int:
		return int64(rv)
	case int8:
		return int64(rv)
	case int16:
		return int64(rv)
	case int32:
		return int64(rv)
	case int64:
		return rv

	// These types can be safely converted to signed integers.
	case uint8:
		return uint64(rv)
	case uint16:
		return uint64(rv)
	case uint32:
		return uint64(rv)
	case uint64:
		return v

	case float32:
		return float64(rv)
	case float64:
		return rv

	case string:
		return v
	}
	return v
}

func index(expr ast.Node, out *[]string) {
	switch n := expr.(type) {
	case *ast.BinaryExpr:
		index(n.X, out)
		index(n.Y, out)
	case *ast.ParenExpr:
		index(n.X, out)
	case *ast.Ident:
		if n.Name != "nil" && n.Name != "true" && n.Name != "false" {
			*out = append(*out, n.Name)
		}
	}
}

func eval(value V, expr ast.Node) interface{} {
	switch n := expr.(type) {
	case *ast.BinaryExpr:
		ll := normalize(eval(value, n.X))

		switch n.Op {
		case token.LAND:
			return boolCast(ll) && boolCast(eval(value, n.Y))
		case token.LOR:
			return boolCast(ll) || boolCast(eval(value, n.Y))
		}

		// Bool is first, to support short-circuit evaluation.
		if l, ok := ll.(bool); ok {
			switch n.Op {
			case token.EQL:
				r := boolCast(eval(value, n.Y))
				return l == r
			case token.NEQ:
				r := boolCast(eval(value, n.Y))
				return l != r
			}
			panic(fmt.Errorf("unsupported boolean operation"))
		}

		rr := normalize(eval(value, n.Y))

		if ll == nil {
			switch n.Op {
			case token.EQL:
				return ll == rr
			case token.NEQ:
				return ll != rr
			}
		}

		if rr == nil {
			switch n.Op {
			case token.EQL:
				return ll == nil
			case token.NEQ:
				return ll != nil
			}
		} else {
			rr = normalize(rr)
		}

		switch l := ll.(type) {
		case int64:
			r := intCast(rr)
			switch n.Op {
			case token.EQL:
				return l == r
			case token.NEQ:
				return l != r
			case token.LSS:
				return l < r
			case token.GTR:
				return l > r
			case token.LEQ:
				return l <= r
			case token.GEQ:
				return l >= r

			case token.ADD:
				return l + r
			case token.SUB:
				return l - r
			case token.MUL:
				return l * r
			case token.QUO:
				return l / r
			case token.REM:
				return l % r
			case token.SHL:
				if r < 0 {
					panic(fmt.Errorf("negative shift count"))
				}
				return l << uint64(r)
			case token.SHR:
				if r < 0 {
					panic(fmt.Errorf("negative shift count"))
				}
				return l >> uint64(r)

			case token.AND:
				return l & r
			case token.OR:
				return l | r
			case token.XOR:
				return l ^ r
			case token.AND_NOT:
				return l &^ r
			}

		case uint64:
			r := uintCast(rr)
			switch n.Op {
			case token.EQL:
				return l == r
			case token.NEQ:
				return l != r
			case token.LSS:
				return l < r
			case token.GTR:
				return l > r
			case token.LEQ:
				return l <= r
			case token.GEQ:
				return l >= r

			case token.ADD:
				return l + r
			case token.SUB:
				return l - r
			case token.MUL:
				return l * r
			case token.QUO:
				return l / r
			case token.REM:
				return l % r
			case token.SHL:
				return l << r
			case token.SHR:
				return l >> r

			case token.AND:
				return l & r
			case token.OR:
				return l | r
			case token.XOR:
				return l ^ r
			case token.AND_NOT:
				return l &^ r
			}

		case string:
			r := stringCast(rr)
			switch n.Op {
			case token.ADD:
				return l + r

			case token.EQL:
				return l == r
			case token.NEQ:
				return l != r
			case token.LSS:
				return l < r
			case token.GTR:
				return l > r
			case token.LEQ:
				return l <= r
			case token.GEQ:
				return l >= r
			}
		case float64:
			r := floatCast(rr)
			switch n.Op {
			case token.ADD:
				return l + r
			case token.SUB:
				return l - r
			case token.MUL:
				return l * r
			case token.QUO:
				return l / r

			case token.EQL:
				return l == r
			case token.NEQ:
				return l != r
			case token.LSS:
				return l < r
			case token.GTR:
				return l > r
			case token.LEQ:
				return l <= r
			case token.GEQ:
				return l >= r
			}
		default:
			if ll == nil {
				return nil
			}
			kind := reflect.TypeOf(ll).Kind()
			if kind == reflect.Map {
				return ll
			}
			panic(fmt.Errorf("unsupported type %#v", ll))
		}
		panic(fmt.Errorf("unsupported expression %v %s %v", ll, n.Op, rr))
	case *ast.BasicLit:
		switch n.Kind {
		case token.STRING:
			s, err := strconv.Unquote(n.Value)
			if err != nil {
				panic(err.Error())
			}
			return s
		case token.INT:
			nu, err := strconv.ParseInt(n.Value, 10, 64)
			if err != nil {
				panic(err.Error())
			}
			return nu
		case token.FLOAT:
			n, err := strconv.ParseFloat(n.Value, 64)
			if err != nil {
				panic(err.Error())
			}
			return n
		}
		panic(fmt.Errorf("unsupported type"))
	case *ast.ParenExpr:
		return eval(value, n.X)
	case *ast.UnaryExpr:
		if n.Op == token.NOT {
			return !boolCast(eval(value, n.X))
		}
		panic(fmt.Errorf("unsupported unary operator %s", n.Op))
	case *ast.Ident:
		if v, ok := value[n.Name]; ok {
			return normalize(v)
		}
		if n.Name == "true" {
			return true
		} else if n.Name == "false" {
			return false
		}
		return nil
	case *ast.SelectorExpr:
		v := eval(value, n.X)
		if m, ok := v.(V); ok {
			if v, ok := m[n.Sel.Name]; ok {
				return v
			}
		}
		panic(fmt.Errorf("unknown attribute \"%s\" on %#v", n.Sel.Name, v))
	}
	panic(fmt.Errorf("unsupported expression node %#v", expr))
}

func intCast(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch rv := normalize(v).(type) {
	case int64:
		return rv
	case uint64:
		return int64(rv)
	case float64:
		return int64(rv)
	case bool:
		if rv {
			return 1
		}
		return 0
	default:
		panic(fmt.Errorf("not castable to an int64"))
	}
}

func uintCast(v interface{}) uint64 {
	if v == nil {
		return 0
	}
	switch rv := normalize(v).(type) {
	case int64:
		return uint64(rv)
	case uint64:
		return rv
	case float64:
		return uint64(rv)
	case bool:
		if rv {
			return 1
		}
		return 0
	default:
		panic(fmt.Errorf("not castable to an uint64"))
	}
}

func floatCast(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch rv := normalize(v).(type) {
	case int64:
		return float64(rv)
	case uint64:
		return float64(rv)
	case float64:
		return rv
	case bool:
		if rv {
			return 1.0
		}
		return 0.0
	default:
		panic(fmt.Errorf("not castable to a float64"))
	}
}

func boolCast(v interface{}) bool {
	if v == nil {
		return false
	}
	switch rv := normalize(v).(type) {
	case bool:
		return rv
	case string:
		return rv != ""
	case int64:
		return rv != 0
	case uint64:
		return rv != 0
	case float64:
		return rv != 0.0
	default:
		panic(fmt.Errorf("unsupported boolean value"))
	}
}

func stringCast(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
