# Runtime evaluation of Go-like expressions

# What?

eg.

```go
e := expr.MustCompile("a + 1 > 2")
e.Bool(expr.V{"a": 0}) == false
e.Bool(expr.V{"a": 1}) == false
e.Bool(expr.V{"a": 2}) == true
```

and

```go
e = expr.MustCompile("(a + 2) * 10")
n, err := e.Eval(expr.V{"a": 1})
n.(int64) == 30
```

It *does not* support:

- Function calls (though this would be useful).
- Slices or arrays.

It *does* support:

- Truthiness evaluation, ala Python: eg. given `V{"a": 1}`, evaluating `a` will be true.
- Automatic type coercion. eg. `"a" + 10` == `"a10"`
- Nested fields. eg. `A.B == 2`

# Why?

If you want to add simple expression-based filtering or numeric evaluation,
this might fit your needs.

# Where?

## Install

```sh
$ go get github.com/alecthomas/expr
```

## Docs

You can find the API documentation on [godoc](http://godoc.org/github.com/alecthomas/expr).
