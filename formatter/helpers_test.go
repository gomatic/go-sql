package formatter

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// strNode wraps a string value up as a pg_query node.
func strNode(s string) *pg_query.Node {
	return &pg_query.Node{Node: &pg_query.Node_String_{String_: &pg_query.String{Sval: s}}}
}

// intNode wraps an integer value up as a pg_query node.
func intNode(i int32) *pg_query.Node {
	return &pg_query.Node{Node: &pg_query.Node_Integer{Integer: &pg_query.Integer{Ival: i}}}
}

// aconstIntNode wraps an integer constant expression up as a pg_query node.
func aconstIntNode(i int32) *pg_query.Node {
	return &pg_query.Node{Node: &pg_query.Node_AConst{AConst: &pg_query.A_Const{Val: &pg_query.A_Const_Ival{Ival: &pg_query.Integer{Ival: i}}}}}
}

// columnRefNode wraps a single-field column reference up as a pg_query node.
func columnRefNode(name string) *pg_query.Node {
	return &pg_query.Node{Node: &pg_query.Node_ColumnRef{ColumnRef: &pg_query.ColumnRef{Fields: []*pg_query.Node{strNode(name)}}}}
}

// defElemNode wraps a DefElem up as a pg_query node.
func defElemNode(elem *pg_query.DefElem) *pg_query.Node {
	return &pg_query.Node{Node: &pg_query.Node_DefElem{DefElem: elem}}
}
