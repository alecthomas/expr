package expr

import (
	"testing"

	"github.com/stretchrcom/testify/assert"
)

func TestMatchInt(t *testing.T) {
	value := V{"I": 5}
	assert.True(t, MustCompile("I == 5").Bool(value))
}

func TestBitOps(t *testing.T) {
	value := V{"I": 3}
	assert.True(t, MustCompile("I & 2 == 2").Bool(value))
	assert.True(t, MustCompile("I | 4 == 7").Bool(value))
}

func TestMatchNil(t *testing.T) {
	value := V{}
	assert.True(t, MustCompile(`I == nil`).Bool(value))
	assert.False(t, MustCompile(`I != nil`).Bool(value))
}

func TestMatchShortCircuit(t *testing.T) {
	value := V{}
	assert.True(t, MustCompile(`true || false`).Bool(value))
}

func TestMatchMap(t *testing.T) {
	value := V{"Foo": V{"Bar": "Waz"}}
	assert.True(t, MustCompile(`Foo.Bar == "Waz"`).Bool(value))
}

func TestMissingKey(t *testing.T) {
	assert.True(t, MustCompile(`!Foo`).Bool(nil))
}

func TestMatchNot(t *testing.T) {
	assert.True(t, MustCompile(`!Foo`).Bool(V{"Foo": false}))
}

func TestMatchUnary(t *testing.T) {
	assert.True(t, MustCompile("I").Bool(V{"I": 1}))
	assert.False(t, MustCompile("I").Bool(V{"I": 0}))
}

func TestSubscript(t *testing.T) {
	expr, err := Compile("I[0]")
	assert.NoError(t, err)
	_, err = expr.Eval(V{"I": true})
	assert.Error(t, err)
}

func TestEvaluateAdd(t *testing.T) {
	n, err := MustCompile("3 + 4").Eval(V{})
	assert.NoError(t, err)
	assert.Equal(t, 7, n)
}

func TestEvaluateSub(t *testing.T) {
	n, err := MustCompile("3 - 4").Eval(V{})
	assert.NoError(t, err)
	assert.Equal(t, -1, n)
}

func TestEvaluateComplex(t *testing.T) {
	n, err := MustCompile("3 + 4 * 2 / (1 - 5) + 3").Eval(V{})
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
}

func TestEvaluateStringCoercion(t *testing.T) {
	n, err := MustCompile(`"foo" + 10`).Eval(V{})
	assert.NoError(t, err)
	assert.Equal(t, "foo10", n)
}

func TestTruthinessLogical(t *testing.T) {
	n := MustCompile("a && b").Bool(V{"a": 1, "b": 1})
	assert.True(t, n)
}

func TestTerms(t *testing.T) {
	e := MustCompile("a + b * c + c")
	assert.Equal(t, []string{"a", "b", "c", "c"}, e.Terms)
}

func BenchmarkMatching(b *testing.B) {
	value := V{"I": 5}
	m := MustCompile("I == 5 || I == 3")
	for i := 0; i < b.N; i++ {
		m.Bool(value)
	}
}

func BenchmarkEval(t *testing.B) {
	expr := MustCompile("3 + 4 * 2 / (1 - 5) + 3")
	cx := V{}
	for i := 0; i < t.N; i++ {
		_, err := expr.Eval(cx)
		if err != nil {
			panic(err)
		}
	}
}

func TestStringCast(t *testing.T) {
	assert.Equal(t, "10", stringCast(int64(10)))
	assert.Equal(t, "10.5", stringCast(float64(10.5)))
	assert.Equal(t, "true", stringCast(true))
	assert.Equal(t, "", stringCast(nil))
	assert.Equal(t, "hello", stringCast("hello"))
}
