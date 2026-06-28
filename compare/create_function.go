package compare

import (
	"strings"
)

type (
	argTypesString string // argTypesString is a comma-separated list of argument types.
	typeName       string // typeName is a database type name.
)

// createFunctionIdentity identifies a CREATE FUNCTION by qualified name and
// argument types — for example, create.function:schema.fn(int4,text).
func createFunctionIdentity(stmt statementData) identity {
	data := extractStatementData(stmt)
	if data == nil {
		return ""
	}
	funcnameList := extractSlice(data, keyFuncname)
	if len(funcnameList) == 0 {
		return ""
	}
	qualName := formatQualifiedName(extractFunctionName(funcnameList))
	argTypes := extractFunctionArgTypes(data)
	return identity(identityPrefixCreateFunction + identityPrefix(qualName) + "(" + identityPrefix(argTypes) + ")")
}

// createFunctionDiff compares two CREATE FUNCTION statements by their signature
// and skips the function body.
func createFunctionDiff(source, target statementData) statementDiffs {
	return computeDiffs(removeFunctionBody(source), removeFunctionBody(target))
}

// extractFunctionName splits a funcname list into a schema and a function name.
func extractFunctionName(funcnameList []any) (schemaName, objectName) {
	names := extractNodeStringParts(funcnameList)
	switch len(names) {
	case 0:
		return "", ""
	case 1:
		return "", objectName(names[0])
	default:
		return schemaName(names[0]), objectName(names[1])
	}
}

// extractFunctionArgTypes joins a function's parameter types together into a
// signature.
func extractFunctionArgTypes(data statementData) argTypesString {
	parameters := extractSlice(data, "parameters")
	types := make([]string, 0, len(parameters))
	for _, param := range parameters {
		if tn := extractParamTypeName(param); tn != "" {
			types = append(types, string(tn))
		}
	}
	return argTypesString(strings.Join(types, ","))
}

// extractParamTypeName reads the declared type off a function parameter.
func extractParamTypeName(param any) typeName {
	m, ok := param.(map[string]any)
	if !ok {
		return ""
	}
	funcParam := extractMap(extractMap(statementData(m), keyNode), "function_parameter")
	return extractTypeName(extractMap(funcParam, "arg_type"))
}

// extractTypeName joins the parts of a TypeName node into one dotted type name.
func extractTypeName(typeNode statementData) typeName {
	return typeName(strings.Join(extractNodeStringParts(extractSlice(typeNode, keyNames)), "."))
}

// removeFunctionBody returns a deep copy of a CREATE FUNCTION statement with its
// body options stripped out, so two functions compare on signature alone.
func removeFunctionBody(stmt statementData) statementData {
	result := deepCopyMap(stmt)
	if data := extractMap(extractMap(result, keyStmt), keyData); data != nil {
		delete(data, "options")
	}
	return result
}

// deepCopyMap returns a deep, recursive copy of a statement map.
func deepCopyMap(m statementData) statementData {
	if m == nil {
		return nil
	}
	result := make(statementData, len(m))
	for k, v := range m {
		result[k] = deepCopyValue(v)
	}
	return result
}

// deepCopyValue returns a recursive copy of a decoded JSON value. Nested maps
// keep their map[string]any dynamic type, so the extract helpers' type
// assertions still match — just like they do for freshly decoded JSON.
func deepCopyValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return map[string]any(deepCopyMap(val))
	case []any:
		return deepCopySlice(val)
	default:
		return v
	}
}

// deepCopySlice returns a deep, recursive copy of a decoded JSON slice.
func deepCopySlice(s []any) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = deepCopyValue(v)
	}
	return result
}
