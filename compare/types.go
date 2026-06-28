package compare

// Core domain types shared across the comparison engine. We carry every
// statement as decoded JSON (statementData), key it by a derived identity, and
// tag it with its pg_query statement type.
type (
	identity       string         // identity uniquely names a statement across the two scripts.
	pgQueryType    string         // pgQueryType is the pg_query statement type (e.g. "create_stmt").
	statementData  map[string]any // statementData is one statement decoded from normalized JSON.
	statementDiffs []fieldDiff    // statementDiffs is the ordered set of field differences for a statement.
)

// pg_query statement type names. Each value matches the "type" tag the root sql
// package emits for a recognized statement, and keys a handler in the registry.
const (
	typeAlterTable     pgQueryType = "alter_table_stmt"
	typeComment        pgQueryType = "comment_stmt"
	typeCreateFunction pgQueryType = "create_function_stmt"
	typeCreateSchema   pgQueryType = "create_schema_stmt"
	typeCreateTable    pgQueryType = "create_stmt"
	typeCreateTrigger  pgQueryType = "create_trig_stmt"
	typeDrop           pgQueryType = "drop_stmt"
	typeGrant          pgQueryType = "grant_stmt"
	typeIndex          pgQueryType = "index_stmt"
	typeView           pgQueryType = "view_stmt"
)

// identityPrefix is the leading, type-specific piece of an identity string.
type identityPrefix string

// Identity prefixes, one per statement kind. They keep identities of different
// kinds from colliding — say, a table and a view that share a name.
const (
	identityPrefixAlterTable     identityPrefix = "alter.table."
	identityPrefixComment        identityPrefix = "comment."
	identityPrefixCreateFunction identityPrefix = "create.function:"
	identityPrefixCreateIndex    identityPrefix = "create.index:"
	identityPrefixCreateSchema   identityPrefix = "create.schema:"
	identityPrefixCreateTable    identityPrefix = "create.table:"
	identityPrefixCreateTrigger  identityPrefix = "create.trigger:"
	identityPrefixCreateView     identityPrefix = "create.view:"
	identityPrefixDrop           identityPrefix = "drop."
	identityPrefixGrant          identityPrefix = "grant."
	identityPrefixRevoke         identityPrefix = "revoke."
)

// fieldKey names a key inside a decoded statement map. Navigation keys that get
// reused across handlers live here, so each string is written down exactly once.
type fieldKey string

const (
	keyAlterTableCmd fieldKey = "alter_table_cmd"
	keyColumnDef     fieldKey = "column_def"
	keyComment       fieldKey = "comment"
	keyConstraint    fieldKey = "constraint"
	keyData          fieldKey = "data"
	keyDef           fieldKey = "def"
	keyFuncname      fieldKey = "funcname"
	keyList          fieldKey = "list"
	keyNames         fieldKey = "names"
	keyNode          fieldKey = "node"
	keyObjWithArgs   fieldKey = "object_with_args"
	keyObjargs       fieldKey = "objargs"
	keyObjects       fieldKey = "objects"
	keyObjname       fieldKey = "objname"
	keyRangeVar      fieldKey = "range_var"
	keyRelation      fieldKey = "relation"
	keyRelname       fieldKey = "relname"
	keyRoleSpec      fieldKey = "role_spec"
	keySchemaname    fieldKey = "schemaname"
	keyStmt          fieldKey = "stmt"
	keyStringNode    fieldKey = "string_"
	keySval          fieldKey = "sval"
	keyType          fieldKey = "type"
	keyTypeName      fieldKey = "type_name"
)

// fieldDiff is a single field that differs between a source and target
// statement.
type fieldDiff struct {
	Source fieldValue
	Target fieldValue
	Field  fieldPath
}

type (
	fieldPath  string // fieldPath is the dot/bracket-separated path to a field in the AST.
	fieldValue any    // fieldValue is the value of a field in the AST.
)

// indexedStatement pairs a statement with where it sat in its source script.
type indexedStatement struct {
	stmt  statementData
	index stmtIndex
}

type (
	stmtIndex int                           // stmtIndex is a statement's position within its script.
	stmtsByID map[identity]indexedStatement // stmtsByID indexes statements by identity.
)
