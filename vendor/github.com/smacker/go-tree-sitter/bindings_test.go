package sitter_test

import (
	"runtime"
	"testing"
	"time"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/stretchr/testify/assert"
)

func TestRootNode(t *testing.T) {
	assert := assert.New(t)

	n := sitter.Parse([]byte("let a = 1"), javascript.GetLanguage())

	assert.Equal(uint32(0), n.StartByte())
	assert.Equal(uint32(9), n.EndByte())
	assert.Equal(sitter.Point{
		Row:    0,
		Column: 0,
	}, n.StartPoint())
	assert.Equal(sitter.Point{
		Row:    0,
		Column: 9,
	}, n.EndPoint())
	assert.Equal("(program (lexical_declaration (variable_declarator name: (identifier) value: (number))))", n.String())
	assert.Equal("program", n.Type())
	assert.Equal(sitter.Symbol(119), n.Symbol())

	assert.Equal(false, n.IsNull())
	assert.Equal(true, n.IsNamed())
	assert.Equal(false, n.IsMissing())
	assert.Equal(false, n.HasChanges())
	assert.Equal(false, n.HasError())

	assert.Equal(uint32(1), n.ChildCount())
	assert.Equal(uint32(1), n.NamedChildCount())

	assert.Nil(n.Parent())
	assert.Nil(n.NextSibling())
	assert.Nil(n.NextNamedSibling())
	assert.Nil(n.PrevSibling())
	assert.Nil(n.PrevNamedSibling())

	assert.NotNil(n.Child(0))
	assert.NotNil(n.NamedChild(0))
	assert.Nil(n.ChildByFieldName("unknown"))

	assert.NotNil(n.NamedChild(0).NamedChild(0).ChildByFieldName("name"))
}

func TestTree(t *testing.T) {
	assert := assert.New(t)

	parser := sitter.NewParser()

	parser.Debug()
	parser.SetLanguage(javascript.GetLanguage())
	tree := parser.Parse([]byte("let a = 1"))
	n := tree.RootNode()

	assert.Equal(uint32(0), n.StartByte())
	assert.Equal(uint32(9), n.EndByte())
	assert.Equal("program", n.Type())
	assert.Equal("(program (lexical_declaration (variable_declarator name: (identifier) value: (number))))", n.String())

	tree2 := parser.Parse([]byte("let a = 'a'"))
	n = tree2.RootNode()
	assert.Equal("(program (lexical_declaration (variable_declarator name: (identifier) value: (string))))", n.String())

	// change 'a' -> true
	newText := []byte("let a = true")
	tree2.Edit(sitter.EditInput{
		StartIndex:  8,
		OldEndIndex: 11,
		NewEndIndex: 12,
		StartPoint: sitter.Point{
			Row:    0,
			Column: 8,
		},
		OldEndPoint: sitter.Point{
			Row:    0,
			Column: 11,
		},
		NewEndPoint: sitter.Point{
			Row:    0,
			Column: 12,
		},
	})
	// check that it changed tree
	assert.True(n.HasChanges())
	assert.True(n.Child(0).HasChanges())
	assert.False(n.Child(0).Child(0).HasChanges()) // left side of tree didn't change
	assert.True(n.Child(0).Child(1).HasChanges())

	tree3 := parser.ParseWithTree(newText, tree2)
	n = tree3.RootNode()
	assert.Equal("(program (lexical_declaration (variable_declarator name: (identifier) value: (true))))", n.String())
}

func TestLanguage(t *testing.T) {
	assert := assert.New(t)
	js := javascript.GetLanguage()

	assert.True(js.SymbolCount() > 171)
	assert.Equal(js.SymbolName(171), "class_declaration")
	assert.Equal(js.SymbolType(171), sitter.SymbolTypeRegular)

	assert.Equal(sitter.SymbolTypeRegular.String(), "Regular")
}

func TestGC(t *testing.T) {
	assert := assert.New(t)

	parser := sitter.NewParser()

	parser.SetLanguage(javascript.GetLanguage())
	tree := parser.Parse([]byte("let a = 1"))
	n := tree.RootNode()

	r := isNamedWithGC(n)
	assert.True(r)
}

func isNamedWithGC(n *sitter.Node) bool {
	runtime.GC()
	time.Sleep(500 * time.Microsecond)
	return n.IsNamed()
}

func TestSetOperationLimit(t *testing.T) {
	assert := assert.New(t)

	parser := sitter.NewParser()
	assert.Equal(0, parser.OperationLimit())

	parser.SetOperationLimit(10)
	assert.Equal(10, parser.OperationLimit())
}

func TestIncludedRanges(t *testing.T) {
	assert := assert.New(t)

	// go code with JS in comment
	code := "package main\n//console.log('sup');"

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	goTree := parser.Parse([]byte(code))
	commentNode := goTree.RootNode().NamedChild(1)
	assert.Equal("comment", commentNode.Type())

	jsRange := sitter.Range{
		StartPoint: sitter.Point{
			Row:    commentNode.StartPoint().Row,
			Column: commentNode.StartPoint().Column + 2,
		},
		EndPoint:  commentNode.EndPoint(),
		StartByte: commentNode.StartByte() + 2,
		EndByte:   commentNode.EndByte(),
	}

	parser.SetIncludedRanges([]sitter.Range{jsRange})
	parser.SetLanguage(javascript.GetLanguage())
	jsTree := parser.Parse([]byte(code))

	assert.Equal(
		"(program (expression_statement (call_expression function: (member_expression object: (identifier) property: (property_identifier)) arguments: (arguments (string)))))",
		jsTree.RootNode().String(),
	)
}

func TestSameNode(t *testing.T) {
	assert := assert.New(t)

	parser := sitter.NewParser()
	parser.SetLanguage(javascript.GetLanguage())
	tree := parser.Parse([]byte("let a = 1"))

	n1 := tree.RootNode()
	n2 := tree.RootNode()

	assert.True(n1 == n2)

	n1 = tree.RootNode().NamedChild(0)
	n2 = tree.RootNode().NamedChild(0)

	assert.True(n1 == n2)
}

func TestQuery(t *testing.T) {
	js := `
	class Person {
		constructor(firstName, parent) {}
		getName() {}
	}
	`

	// test single capture
	testCaptures(t, js, "(class_declaration name: (identifier) @class-name)", []string{
		"Person",
	})

	// test multiple captures
	testCaptures(t, js, "(method_definition name: * @method-name)", []string{
		"constructor",
		"getName",
	})

	// test match only
	parser := sitter.NewParser()
	parser.SetLanguage(javascript.GetLanguage())
	tree := parser.Parse([]byte(js))
	root := tree.RootNode()

	q, err := sitter.NewQuery([]byte("(identifier) (method_definition)"), javascript.GetLanguage())
	assert.Nil(t, err)

	qc := sitter.NewQueryCursor()
	qc.Exec(q, root)

	var matched int
	for {
		_, ok := qc.NextMatch()
		if !ok {
			break
		}

		matched++
	}

	assert.Equal(t, 5, matched)
}

func testCaptures(t *testing.T, body, sq string, expected []string) {
	assert := assert.New(t)

	parser := sitter.NewParser()
	parser.SetLanguage(javascript.GetLanguage())
	tree := parser.Parse([]byte(body))
	root := tree.RootNode()

	q, err := sitter.NewQuery([]byte(sq), javascript.GetLanguage())
	assert.Nil(err)

	qc := sitter.NewQueryCursor()
	qc.Exec(q, root)

	actual := []string{}
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		for _, c := range m.Captures {
			actual = append(actual, body[c.Node.StartByte():c.Node.EndByte()])
		}
	}

	assert.EqualValues(expected, actual)
}

func TestQueryError(t *testing.T) {
	assert := assert.New(t)

	q, err := sitter.NewQuery([]byte("((unknown) name: (identifier))"), javascript.GetLanguage())

	assert.Nil(q)
	assert.NotNil(err)
	assert.EqualValues(sitter.QueryErrorNodeType{Offset: 0x02}, err)
}
