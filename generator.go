package main

import (
	"fmt"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type Scope struct {
	module        *ir.Module
	ctx           *BlockContext
	ifLeaveBlock  *ir.Block
	forLeaveBlock *ir.Block
	functions     map[string]*FunctionContext
}

func (s *Scope) newChildScope(ctx *BlockContext, forLeaveBlock *ir.Block) *Scope {
	scope := &Scope{module: s.module, ctx: ctx, forLeaveBlock: forLeaveBlock}
	return scope
}

type FunctionContext struct {
	target *ir.Func
}

type BlockContext struct {
	*ir.Block
	parentVariableContext *BlockContext
	parentFunctionContext *BlockContext
	functions             map[string]*FunctionContext
	vars                  map[string]value.Value
}

func (ctx *BlockContext) newFunctionContext(block *ir.Block) *BlockContext {
	fncCtx := &BlockContext{Block: block, functions: make(map[string]*FunctionContext), vars: make(map[string]value.Value)}
	fncCtx.parentFunctionContext = ctx
	return fncCtx
}

func (ctx *BlockContext) newChildContext(block *ir.Block) *BlockContext {
	childCtx := &BlockContext{Block: block, functions: make(map[string]*FunctionContext), vars: make(map[string]value.Value)}
	childCtx.parentVariableContext = ctx
	childCtx.parentFunctionContext = ctx
	return childCtx
}

func (ctx *BlockContext) copyContext(block *ir.Block) *BlockContext {
	newCtx := &BlockContext{Block: block, functions: ctx.functions, vars: ctx.vars}
	newCtx.parentVariableContext = ctx.parentVariableContext
	newCtx.parentFunctionContext = ctx.parentFunctionContext
	return newCtx
}

var blockIndex uint64

func blockName() string {
	name := fmt.Sprintf("block-%d", blockIndex)
	blockIndex++
	return name
}

func GenerateIR(node *Node) string {
	scope := &Scope{}
	module := ir.NewModule()
	scope.module = module
	scope.registerGlobalFuncs()

	mainFunc := module.NewFunc("main", types.I32)
	mainBlock := mainFunc.NewBlock("")

	mainBlockContext := BlockContext{Block: mainBlock, functions: scope.functions, vars: make(map[string]value.Value)}
	scope.ctx = &mainBlockContext
	scope.resolveStmts(node)

	return module.String()
}

func (s *Scope) registerGlobalFuncs() {
	zero := constant.NewInt(types.I64, 0)

	printf := s.module.NewFunc("printf", types.I32, ir.NewParam("format", types.NewPointer(types.I8)))
	printf.Sig.Variadic = true

	printValue := s.module.NewFunc("printValue", types.Void, ir.NewParam("value", types.I32))
	printValueBlock := printValue.NewBlock("")
	integetFormat := s.module.NewGlobalDef(".intF", constant.NewCharArrayFromString("%d\n"))
	message := constant.NewGetElementPtr(integetFormat.ContentType, integetFormat, zero, zero)
	value := printValue.Params[0]
	printValueBlock.NewCall(printf, message, value)
	printValueBlock.NewRet(nil)

	globalFuncs := make(map[string]*FunctionContext)
	printValueContext := &FunctionContext{target: printValue}
	globalFuncs["printValue"] = printValueContext
	s.functions = globalFuncs
}

func (s *Scope) resolveStmts(node *Node) {
	current := node
	for {
		if current == nil {
			break
		}
		s.resolveStmt(current)
		current = current.next
	}
}

func (s *Scope) resolveStmt(node *Node) {
	switch node.nodeType {
	case NodeTypeReturn:
		result := s.ctx.resolveExpr(node.lhs)
		s.ctx.NewRet(result)
	case NodeTypeIF:
		bName := blockName()
		thenBlock := s.ctx.Parent.NewBlock(bName + "-then")
		leaveBlock := s.ctx.Parent.NewBlock(bName + "-leave")
		elseBlock := leaveBlock
		if node.els != nil {
			elseBlock = s.ctx.Parent.NewBlock(bName + "-else")
		}
		s.ctx.NewCondBr(s.ctx.resolveExpr(node.cond), thenBlock, elseBlock)
		thenBlockContext := s.ctx.newChildContext(thenBlock)
		thenScope := s.newChildScope(thenBlockContext, s.forLeaveBlock)
		thenScope.ifLeaveBlock = leaveBlock
		thenScope.resolveStmts(node.then)
		if thenBlock.Term == nil {
			thenBlock.NewBr(leaveBlock)
		}
		if node.els != nil {
			elseBlockContext := s.ctx.newChildContext(elseBlock)
			elseScope := s.newChildScope(elseBlockContext, s.forLeaveBlock)
			elseScope.ifLeaveBlock = leaveBlock
			elseScope.resolveStmts(node.els)
		}
		s.ctx = s.ctx.copyContext(leaveBlock)
		if s.ifLeaveBlock != nil {
			leaveBlock.NewBr(s.ifLeaveBlock)
		}
	case NodeTypeFunction:
		var args []*ir.Param
		for _, arg := range node.argDefs {
			args = append(args, ir.NewParam(arg, types.I32))
		}
		fncName := s.ctx.Parent.Name() + "_" + node.str
		fnc := s.module.NewFunc(fncName, types.I32, args...)
		fncBlock := fnc.NewBlock(blockName())
		fncBlockContext := s.ctx.newFunctionContext(fncBlock)
		fncContext := &FunctionContext{target: fnc}
		s.ctx.functions[node.str] = fncContext
		fncScope := s.newChildScope(fncBlockContext, nil)
		for i, arg := range node.argDefs {
			fncScope.ctx.vars[arg] = fnc.Params[i]
		}
		fncScope.resolveStmts(node.blockBody)
		if fncBlockContext.Term == nil {
			fncBlockContext.NewRet(nil)
		}
	case NodeTypeFunctionCall:
		s.ctx.callFunction(node)
	case NodeTypeAssign:
		v := s.ctx.findVariable(node.lhs.str)
		vr := s.ctx.resolveExpr(node.rhs)
		s.ctx.NewStore(vr, v)
	case NodeTypeAlloc:
		v := s.ctx.NewAlloca(types.I32)
		vr := s.ctx.resolveExpr(node.rhs)
		s.ctx.NewStore(vr, v)
		s.ctx.vars[node.lhs.str] = v
	case NodeTypeFor:
		bName := blockName()
		loopBlock := s.ctx.Parent.NewBlock(bName + "-for")
		leaveBlock := s.ctx.Parent.NewBlock(bName + "-leave-for")
		s.ctx.NewBr(loopBlock)
		loopBlockContext := s.ctx.newChildContext(loopBlock)
		loopScope := s.newChildScope(loopBlockContext, nil)
		loopScope.forLeaveBlock = leaveBlock
		loopScope.resolveStmts(node.blockBody)
		loopScope.ctx.NewBr(loopBlock)
		s.ctx = s.ctx.copyContext(leaveBlock)
	case NodeTypeBreak:
		s.ctx.NewBr(s.forLeaveBlock)
	case NodeTypeBlock:
		current := node.blockBody
		for {
			if current == nil {
				break
			}
			s.resolveStmt(current)
			current = current.next
		}
	default:
		break
	}
}

func (ctx *BlockContext) resolveExpr(node *Node) value.Value {
	switch node.nodeType {
	case NodeTypeSum:
		l, r := ctx.resolveExpr(node.lhs), ctx.resolveExpr(node.rhs)
		return ctx.NewAdd(l, r)
	case NodeTypeSub:
		l, r := ctx.resolveExpr(node.lhs), ctx.resolveExpr(node.rhs)
		return ctx.NewSub(l, r)
	case NodeTypeMul:
		l, r := ctx.resolveExpr(node.lhs), ctx.resolveExpr(node.rhs)
		return ctx.NewMul(l, r)
	case NodeTypeDiv:
		l, r := ctx.resolveExpr(node.lhs), ctx.resolveExpr(node.rhs)
		return ctx.NewSDiv(l, r)
	case NodeTypeEqual:
		l, r := ctx.resolveExpr(node.lhs), ctx.resolveExpr(node.rhs)
		return ctx.NewICmp(enum.IPredEQ, l, r)
	case NodeTypeNotEqual:
		l, r := ctx.resolveExpr(node.lhs), ctx.resolveExpr(node.rhs)
		return ctx.NewICmp(enum.IPredNE, l, r)
	case NodeTypeLT:
		l, r := ctx.resolveExpr(node.lhs), ctx.resolveExpr(node.rhs)
		return ctx.NewICmp(enum.IPredSLT, l, r)
	case NodeTypeLE:
		l, r := ctx.resolveExpr(node.lhs), ctx.resolveExpr(node.rhs)
		return ctx.NewICmp(enum.IPredSLE, l, r)
	case NodeTypeNum:
		return constant.NewInt(types.I32, int64(node.value))
	case NodeTypeLVar:
		v := ctx.findVariable(node.str)
		if v.Type().Equal(types.NewPointer(types.I32)) {
			return ctx.NewLoad(types.I32, v)
		} else {
			return v
		}
	case NodeTypeFunctionCall:
		return ctx.callFunction(node)
	default:
		panic("Invalid Node Type")
	}
}

func (ctx *BlockContext) callFunction(node *Node) value.Value {
	fncName := node.str
	var args []value.Value
	current := node.args
	for {
		if current == nil {
			break
		}
		value := ctx.resolveExpr(current)
		args = append(args, value)
		current = current.next
	}
	fnc := ctx.findFunction(fncName)
	res := ctx.NewCall(fnc, args...)
	return res
}

func (ctx *BlockContext) findVariable(name string) value.Value {
	if v, ok := ctx.vars[name]; ok {
		return v
	} else if ctx.parentVariableContext != nil {
		return ctx.parentVariableContext.findVariable(name)
	} else {
		panic("Not Found Variable")
	}
}

func (ctx *BlockContext) findFunction(name string) *ir.Func {
	if v, ok := ctx.functions[name]; ok {
		return v.target
	} else if ctx.parentFunctionContext != nil {
		return ctx.parentFunctionContext.findFunction(name)
	} else {
		panic("Not Found Function")
	}
}
