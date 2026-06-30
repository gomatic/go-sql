package compare

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sql "github.com/gomatic/go-sql"
)

// Common test types shared across the package's tests.
type (
	diffCount    int    // diffCount is an expected number of diffs.
	expectBool   bool   // expectBool is an expected boolean result.
	expectEqual  bool   // expectEqual is an expected equality result.
	sqlStatement string // sqlStatement is the raw SQL text under test.
	testdataFile string // testdataFile is a filename under testdata/.
	testName     string // testName names a test case.
)

// parseTestSQL parses a single SQL statement into its decoded statement map,
// running it through the same parse → ToJSON → decode pipeline the library uses.
func parseTestSQL(t *testing.T, must *require.Assertions, s sqlStatement) statementData {
	t.Helper()

	result, err := pg_query.Parse(string(s))
	must.NoError(err)
	must.Len(result.Stmts, 1)

	stmts, err := sql.ToJSON(result)
	must.NoError(err)
	must.Len(stmts, 1)

	var stmt statementData
	must.NoError(json.Unmarshal(stmts[0], &stmt))
	return stmt
}

// parseStatement is just parseTestSQL under a more descriptive name.
func parseStatement(t *testing.T, must *require.Assertions, s sqlStatement) statementData {
	t.Helper()
	return parseTestSQL(t, must, s)
}

// parseAndExtractData parses SQL and returns the data payload, checking the type
// along the way.
func parseAndExtractData(
	t *testing.T,
	must *require.Assertions,
	s sqlStatement,
	expectedType pgQueryType,
) statementData {
	t.Helper()

	stmt := parseStatement(t, must, s)
	stmtObj := extractMap(stmt, keyStmt)
	must.NotNil(stmtObj, "statement should have stmt wrapper")
	must.Equal(string(expectedType), string(extractString(stmtObj, keyType)), "unexpected statement type")

	data := extractMap(stmtObj, keyData)
	must.NotNil(data, "statement should have data")
	return data
}

// assertASTContains checks that actual carries every field expected has.
func assertASTContains(t *testing.T, want *assert.Assertions, expected, actual statementData) {
	t.Helper()
	for key, expectedVal := range expected {
		actualVal, exists := actual[key]
		want.True(exists, "expected field %q to exist", key)
		want.Equal(expectedVal, actualVal, "field %q mismatch", key)
	}
}

// sqlFromFile reads a testdata SQL file in as raw SQL text.
func sqlFromFile(t *testing.T, must *require.Assertions, filename testdataFile) sql.SQL {
	t.Helper()
	sqlBytes, err := os.ReadFile(filepath.Join("testdata", string(filename)))
	must.NoError(err, "failed to read test file")
	return sql.SQL(sqlBytes)
}
