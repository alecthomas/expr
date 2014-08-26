# Runtime evaluation of Go-like expressions

eg.

    e := expr.MustCompile("a + 1 > 2")
    e.Bool(expr.V{"a": 0}) == false
    e.Bool(expr.V{"a": 1}) == false
    e.Bool(expr.V{"a": 2}) == true

    e = expr.MustCompile("a + 2 * 10")
    n, err := e.Eval(expr.V{"a": 1})
