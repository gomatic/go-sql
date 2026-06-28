package sql

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToJSONNilResult(t *testing.T) {
	t.Parallel()
	out, err := ToJSON(nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestToJSONRecognizedStatement(t *testing.T) {
	t.Parallel()
	tree, err := Parse("CREATE TABLE t (id int)")
	require.NoError(t, err)

	out, err := ToJSON(tree)
	require.NoError(t, err)
	require.Len(t, out, 1)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(out[0], &decoded))
	stmt, ok := decoded["stmt"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "create_stmt", stmt["type"])
	assert.Contains(t, stmt, "data")
}

func TestToJSONUnrecognizedStatementUsesReflectiveFallback(t *testing.T) {
	t.Parallel()
	// SET is a VariableSetStmt, which isn't in recognizedStmts, so it goes down
	// the reflective default path.
	tree, err := Parse("SET search_path = public")
	require.NoError(t, err)

	out, err := ToJSON(tree)
	require.NoError(t, err)
	require.Len(t, out, 1)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(out[0], &decoded))
	stmt := decoded["stmt"].(map[string]any)
	assert.Equal(t, "variable_set_stmt", stmt["type"])
}

func TestToJSONMultipleStatements(t *testing.T) {
	t.Parallel()
	tree, err := Parse("CREATE TABLE a (id int); CREATE TABLE b (id int)")
	require.NoError(t, err)

	out, err := ToJSON(tree)
	require.NoError(t, err)
	assert.Len(t, out, 2)
}

func TestToJSONMarshalFailureWrapsErrMarshal(t *testing.T) {
	t.Parallel()
	tree, err := Parse("CREATE TABLE t (id int)")
	require.NoError(t, err)

	boom := errors.New("boom")
	_, err = toJSON(tree, func(any) ([]byte, error) { return nil, boom })
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMarshal)
	assert.ErrorIs(t, err, boom)
}

func TestNormalizeStatementNil(t *testing.T) {
	t.Parallel()
	assert.Equal(t, map[string]any{}, normalizeStatement(nil))
	assert.Equal(t, map[string]any{}, normalizeStatement(&pg_query.RawStmt{}))
}

func TestNormalizeNodeNil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, normalizeNode(nil))
}

func TestNormalizeDefaultNonPointerOmitsData(t *testing.T) {
	t.Parallel()
	// A non-pointer argument hits the branch that records only the type.
	got := normalizeDefault(42)
	assert.Equal(t, "int", got["type"])
	assert.NotContains(t, got, "data")
}

func TestNormalizeDefaultNilPointerOmitsData(t *testing.T) {
	t.Parallel()
	var p *pg_query.SelectStmt
	got := normalizeDefault(p)
	assert.NotContains(t, got, "data")
}

func TestNodeTypeNameStripsQualifierAndPrefix(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "do_stmt", nodeTypeName(&pg_query.Node_DoStmt{}))
	assert.Equal(t, "int", nodeTypeName(0))
}

func TestNormalizeStructValueInvalid(t *testing.T) {
	t.Parallel()
	assert.Nil(t, normalizeStructValue(reflect.Value{}))
}

func TestNormalizeStructValueNilPointer(t *testing.T) {
	t.Parallel()
	var p *pg_query.SelectStmt
	assert.Nil(t, normalizeStructValue(reflect.ValueOf(p)))
}

func TestNormalizeStructValuePointerIsDereferenced(t *testing.T) {
	t.Parallel()
	got := normalizeStructValue(reflect.ValueOf(&struct{ Name string }{Name: "x"}))
	assert.Equal(t, map[string]any{"name": "x"}, got)
}

func TestNormalizeStructValueNonStructIsPrimitive(t *testing.T) {
	t.Parallel()
	got := normalizeStructValue(reflect.ValueOf("hello"))
	assert.Equal(t, map[string]any{"value": "hello"}, got)
}

func TestNormalizeStructValueSortsAndPrunesZeroFields(t *testing.T) {
	t.Parallel()
	in := struct {
		Beta  string
		Empty string
		Alpha int64
	}{Beta: "b", Empty: "", Alpha: 7}
	got := normalizeStructValue(reflect.ValueOf(in))
	assert.Equal(t, map[string]any{"alpha": int64(7), "beta": "b"}, got)
	assert.NotContains(t, got, "empty")
}

func TestNormalizeValueInvalid(t *testing.T) {
	t.Parallel()
	assert.Nil(t, normalizeValue(reflect.Value{}))
}

func TestNormalizeValueNilPointerAndInterface(t *testing.T) {
	t.Parallel()
	var p *int
	assert.Nil(t, normalizeValue(reflect.ValueOf(p)))
	var i any
	assert.Nil(t, normalizeValue(reflect.ValueOf(&i).Elem()))
}

func TestNormalizeValueComposites(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []any{int64(1), int64(2)}, normalizeValue(reflect.ValueOf([]int{1, 2})))
	assert.Equal(t, []any{int64(1)}, normalizeValue(reflect.ValueOf([2]int{1, 0})))
	assert.Equal(t, map[string]any{"k": "v"}, normalizeValue(reflect.ValueOf(map[string]string{"k": "v"})))
	assert.Equal(t, map[string]any{"name": "x"}, normalizeValue(reflect.ValueOf(struct{ Name string }{Name: "x"})))
}

func TestScalarValueEachKind(t *testing.T) {
	t.Parallel()
	assert.Equal(t, true, scalarValue(reflect.ValueOf(true)))
	assert.Equal(t, int64(-3), scalarValue(reflect.ValueOf(int32(-3))))
	assert.Equal(t, uint64(5), scalarValue(reflect.ValueOf(uint8(5))))
	assert.Equal(t, 2.5, scalarValue(reflect.ValueOf(2.5)))
	assert.Equal(t, "s", scalarValue(reflect.ValueOf("s")))
	// A kind we don't handle otherwise (complex) falls through to Interface().
	assert.Equal(t, complex(1, 2), scalarValue(reflect.ValueOf(complex(1, 2))))
}

func TestNormalizeSliceNil(t *testing.T) {
	t.Parallel()
	var s []int
	assert.Nil(t, normalizeSlice(reflect.ValueOf(s)))
}

func TestNormalizeMapNil(t *testing.T) {
	t.Parallel()
	var m map[string]int
	assert.Nil(t, normalizeMap(reflect.ValueOf(m)))
}

func TestNormalizeMapSortsKeys(t *testing.T) {
	t.Parallel()
	got := normalizeMap(reflect.ValueOf(map[string]int{"b": 2, "a": 1}))
	assert.Equal(t, map[string]any{"a": int64(1), "b": int64(2)}, got)
}

func TestNormalizePrimitiveValueInvalid(t *testing.T) {
	t.Parallel()
	assert.Nil(t, normalizePrimitiveValue(reflect.Value{}))
}

func TestIsZeroValue(t *testing.T) {
	t.Parallel()
	assert.True(t, isZeroValue(nil))
	assert.True(t, isZeroValue([]any{}))
	assert.True(t, isZeroValue(map[string]any{}))
	assert.True(t, isZeroValue(""))
	assert.True(t, isZeroValue((*int)(nil)))
	assert.True(t, isZeroValue(int64(0)))
	assert.False(t, isZeroValue(int64(1)))
	assert.False(t, isZeroValue("x"))
}

func TestToSnakeCase(t *testing.T) {
	t.Parallel()
	assert.Empty(t, toSnakeCase(""))
	assert.Equal(t, "create_table_stmt", toSnakeCase("CreateTableStmt"))
	assert.Equal(t, "id", toSnakeCase("ID"))
	assert.Equal(t, "lower", toSnakeCase("lower"))
}
