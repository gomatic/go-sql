package compare

import (
	"encoding/json"

	errs "github.com/gomatic/go-error"
	pg_query "github.com/pganalyze/pg_query_go/v6"

	sql "github.com/gomatic/go-sql"
)

// Sentinel errors this package can return. Match them with [errors.Is], not by
// string. A parse failure comes straight through as the root package's
// [sql.ErrParse], unwrapped.
const (
	// ErrConvert means we couldn't render a parsed AST to JSON.
	ErrConvert errs.Const = "convert AST"
	// ErrDecode means we couldn't decode a normalized statement.
	ErrDecode errs.Const = "decode statement"
)

// jsonEncoder is [sql.ToJSON]'s signature: it turns an AST into one normalized
// JSON message per statement. We inject it so a test can drive the
// conversion-failure path.
type jsonEncoder func(*pg_query.ParseResult) ([]json.RawMessage, error)

// jsonDecoder is [json.Unmarshal]'s signature: it decodes one JSON message into
// a value. We inject it so a test can drive the decode-failure path.
type jsonDecoder func(data []byte, v any) error

// Compare parses source and target SQL and tells you what changed at the
// statement level: which statements were added, removed, or changed going from
// source to target. A parse failure comes back as [sql.ErrParse]. Finding
// differences isn't an error — look at the returned [Result].
func Compare(source, target sql.SQL) (Result, error) {
	return compareWith(sql.ToJSON, json.Unmarshal, source, target)
}

// compareWith is Compare with its JSON conversion seams injected.
func compareWith(encode jsonEncoder, decode jsonDecoder, source, target sql.SQL) (Result, error) {
	src, err := statements(encode, decode, source)
	if err != nil {
		return Result{}, err
	}
	tgt, err := statements(encode, decode, target)
	if err != nil {
		return Result{}, err
	}
	reg := newRegistry()
	return diffStatements(reg, indexStatements(reg, src), indexStatements(reg, tgt)), nil
}

// statements parses one script and decodes it into a slice of normalized
// statement maps.
func statements(encode jsonEncoder, decode jsonDecoder, script sql.SQL) ([]statementData, error) {
	tree, err := sql.Parse(script)
	if err != nil {
		return nil, err
	}
	messages, err := encode(tree)
	if err != nil {
		return nil, ErrConvert.With(err)
	}
	return decodeStatements(decode, messages)
}

// decodeStatements decodes each normalized JSON message into its own statement
// map.
func decodeStatements(decode jsonDecoder, messages []json.RawMessage) ([]statementData, error) {
	stmts := make([]statementData, 0, len(messages))
	for _, message := range messages {
		var stmt statementData
		if err := decode(message, &stmt); err != nil {
			return nil, ErrDecode.With(err)
		}
		stmts = append(stmts, stmt)
	}
	return stmts, nil
}
