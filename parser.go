package main

import (
	"fmt"
)

type NodeType int

const (
	NodeTypeSum NodeType = iota
	NodeTypeSub
	NodeTypeMul
	NodeTypeDiv
	NodeTypeEqual
	NodeTypeNotEqual
	NodeTypeLT
	NodeTypeLE
	NodeTypeReturn
	NodeTypeAlloc
	NodeTypeAssign
	NodeTypeLVar
	NodeTypeIF
	NodeTypeBlock
	NodeTypeNum
	NodeTypeFor
	NodeTypeBreak
	NodeTypeFunction
	NodeTypeFunctionCall
)

type Node struct {
	nodeType  NodeType
	lhs       *Node
	rhs       *Node
	value     int
	str       string
	cond      *Node
	then      *Node
	els       *Node
	next      *Node
	blockBody *Node
	args      *Node
	argDefs   []string
}

type ParserContext struct {
	currentToken *Token
}

func (ctx *ParserContext) Parse() *Node {
	node := ctx.program()
	return node
}

func (ctx *ParserContext) program() *Node {
	head := ctx.stmt()
	current := head
	for {
		if ctx.currentToken.tokenType == TokenTypeEof {
			break
		}
		current.next = ctx.stmt()
		current = current.next
	}
	return head
}

func (ctx *ParserContext) stmt() *Node {
	if ctx.consume("fnc") {
		name := ctx.expectIdentifier()
		ctx.expect("(")
		argDefs := ctx.argDefs()
		node := &Node{nodeType: NodeTypeFunction, argDefs: argDefs}
		node.str = name
		node.blockBody = ctx.stmt()
		return node
	}
	if ctx.consume("return") {
		node := &Node{nodeType: NodeTypeReturn}
		node.lhs = ctx.assign()
		ctx.expect(";")
		return node
	}
	if ctx.consume("if") {
		node := &Node{nodeType: NodeTypeIF}
		ctx.expect("(")
		node.cond = ctx.expr()
		ctx.expect(")")
		node.then = ctx.stmt()
		if ctx.consume("else") {
			node.els = ctx.stmt()
		}
		return node
	}
	if ctx.consume("for") {
		node := &Node{nodeType: NodeTypeFor}
		node.blockBody = ctx.stmt()
		return node
	}
	if ctx.consume("break") {
		node := &Node{nodeType: NodeTypeBreak}
		ctx.expect(";")
		return node
	}
	if ctx.consume("{") {
		var head *Node
		var current *Node
		for {
			if ctx.consume("}") {
				break
			}
			if current != nil {
				current.next = ctx.stmt()
				current = current.next
			} else {
				current = ctx.stmt()
				head = current
			}
		}
		node := &Node{nodeType: NodeTypeBlock}
		node.blockBody = head
		return node
	}
	if ctx.consume("var") {
		node := ctx.expr()
		if ctx.consume("=") {
			binary := Node{nodeType: NodeTypeAlloc}
			binary.lhs = node
			binary.rhs = ctx.expr()
			node = &binary
		}
		ctx.expect(";")
		return node
	}
	node := ctx.assign()
	ctx.expect(";")
	return node
}

func (ctx *ParserContext) assign() *Node {
	node := ctx.expr()
	if ctx.consume("=") {
		binary := Node{nodeType: NodeTypeAssign}
		binary.lhs = node
		binary.rhs = ctx.expr()
		node = &binary
	}
	return node
}

func (ctx *ParserContext) expr() *Node {
	return ctx.equality()
}

func (ctx *ParserContext) equality() *Node {
	node := ctx.relational()
	for {
		if ctx.consume("==") {
			relational := ctx.relational()
			binary := &Node{nodeType: NodeTypeEqual}
			binary.lhs = node
			binary.rhs = relational
			node = binary
		} else if ctx.consume("!=") {
			relational := ctx.relational()
			binary := &Node{nodeType: NodeTypeNotEqual}
			binary.lhs = node
			binary.rhs = relational
			node = binary
		} else {
			break
		}
	}
	return node
}

func (ctx *ParserContext) relational() *Node {
	node := ctx.add()
	for {
		if ctx.consume("<") {
			add := ctx.add()
			binary := &Node{nodeType: NodeTypeLT}
			binary.lhs = node
			binary.rhs = add
			node = binary
		} else if ctx.consume("<=") {
			add := ctx.add()
			binary := &Node{nodeType: NodeTypeLE}
			binary.lhs = node
			binary.rhs = add
			node = binary
		} else if ctx.consume(">") {
			add := ctx.add()
			binary := &Node{nodeType: NodeTypeLT}
			binary.lhs = add
			binary.rhs = node
			node = binary
		} else if ctx.consume(">=") {
			add := ctx.add()
			binary := &Node{nodeType: NodeTypeLE}
			binary.lhs = add
			binary.rhs = node
			node = binary
		} else {
			break
		}
	}
	return node
}

func (ctx *ParserContext) add() *Node {
	node := ctx.mul()
	for {
		if ctx.consume("+") {
			mul := ctx.mul()
			binary := &Node{nodeType: NodeTypeSum}
			binary.lhs = node
			binary.rhs = mul
			node = binary
		} else if ctx.consume("-") {
			mul := ctx.mul()
			binary := &Node{nodeType: NodeTypeSub}
			binary.lhs = node
			binary.rhs = mul
			node = binary
		} else {
			break
		}
	}
	return node
}

func (ctx *ParserContext) mul() *Node {
	node := ctx.unary()
	for {
		if ctx.consume("*") {
			pri := ctx.unary()
			binary := &Node{nodeType: NodeTypeMul}
			binary.lhs = node
			binary.rhs = pri
			node = binary
		} else if ctx.consume("/") {
			pri := ctx.unary()
			binary := &Node{nodeType: NodeTypeDiv}
			binary.lhs = node
			binary.rhs = pri
			node = binary
		} else {
			break
		}
	}
	return node
}

func (ctx *ParserContext) unary() *Node {
	if ctx.consume("+") {
		return ctx.primary()
	} else if ctx.consume("-") {
		node := &Node{nodeType: NodeTypeSub}
		node.lhs = &Node{nodeType: NodeTypeNum, value: 0}
		node.rhs = ctx.primary()
		return node
	} else {
		return ctx.primary()
	}
}

func (ctx *ParserContext) primary() *Node {
	if ctx.consume("(") {
		node := ctx.expr()
		ctx.expect(")")
		return node
	}
	identifier := ctx.consumeIdentifier()
	if identifier != "" {
		if ctx.consume("(") {
			args := ctx.args()
			node := &Node{nodeType: NodeTypeFunctionCall, str: identifier, args: args}
			return node
		} else {
			node := &Node{nodeType: NodeTypeLVar, str: identifier}
			return node
		}
	}
	num := ctx.expectNumber()
	node := &Node{nodeType: NodeTypeNum, value: num}
	return node
}

func (ctx *ParserContext) argDefs() []string {
	if ctx.consume(")") {
		return nil
	}
	var defs []string
	identifier := ctx.consumeIdentifier()
	defs = append(defs, identifier)
	for {
		if ctx.consume(",") {
			identifier := ctx.consumeIdentifier()
			defs = append(defs, identifier)
		} else {
			break
		}
	}
	ctx.expect(")")
	return defs
}

func (ctx *ParserContext) args() *Node {
	if ctx.consume(")") {
		return nil
	}
	head := ctx.expr()
	current := head
	for {
		if ctx.consume(",") {
			current.next = ctx.expr()
			current = current.next
		} else {
			break
		}
	}
	ctx.expect(")")
	return head
}

// -- Helper

func (ctx *ParserContext) consume(str string) bool {
	if ctx.currentToken == nil {
		return false
	}
	token := ctx.currentToken
	if token.tokenType == TokenTypeReserved && token.str == str {
		ctx.currentToken = token.next
		return true
	}
	return false
}

func (ctx *ParserContext) consumeIdentifier() string {
	if ctx.currentToken == nil {
		return ""
	}
	token := ctx.currentToken
	if token.tokenType == TokenTypeIdentifier {
		ctx.currentToken = token.next
		return token.str
	}
	return ""
}

func (ctx *ParserContext) expect(str string) {
	if ctx.currentToken == nil {
		panic("invalid token")
	}
	token := ctx.currentToken
	if token.tokenType == TokenTypeReserved && token.str == str {
		ctx.currentToken = token.next
		return
	}
	fmt.Println(str)
	panic("expected token")
}

func (ctx *ParserContext) expectNumber() int {
	if ctx.currentToken == nil {
		panic("invalid token")
	}
	token := ctx.currentToken
	if token.tokenType == TokenTypeNumber {
		ctx.currentToken = token.next
		return token.value
	}
	panic("expected number")
}

func (ctx *ParserContext) expectIdentifier() string {
	if ctx.currentToken == nil {
		panic("invalid token")
	}
	token := ctx.currentToken
	if token.tokenType == TokenTypeIdentifier {
		ctx.currentToken = token.next
		return token.str
	}
	panic("expected identifier")
}
