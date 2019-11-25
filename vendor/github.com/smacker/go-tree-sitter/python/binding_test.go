package python_test

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/stretchr/testify/assert"
)

func TestGrammar(t *testing.T) {
	assert := assert.New(t)

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	sourceCode := []byte("print(1)")
	tree := parser.Parse(sourceCode)

	assert.Equal(
		"(module (expression_statement (call (identifier) (argument_list (integer)))))",
		tree.RootNode().String(),
	)
}
