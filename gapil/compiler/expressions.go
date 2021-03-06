// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compiler

import (
	"fmt"
	"strings"

	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

func (c *C) expression(s *S, e semantic.Expression) *codegen.Value {
	old := c.setCurrentExpression(s, e)
	var out *codegen.Value
	switch e := e.(type) {
	case *semantic.ArrayIndex:
		out = c.arrayIndex(s, e)
	case *semantic.ArrayInitializer:
		out = c.arrayInitializer(s, e)
	case *semantic.BinaryOp:
		out = c.binaryOp(s, e)
	case *semantic.BitTest:
		out = c.bitTest(s, e)
	case semantic.BoolValue:
		out = c.boolValue(s, e)
	case *semantic.Call:
		out = c.call(s, e)
	case *semantic.Cast:
		out = c.cast(s, e)
	case *semantic.ClassInitializer:
		out = c.classInitializer(s, e)
	case *semantic.Create:
		out = c.create(s, e)
	case *semantic.Clone:
		out = c.clone(s, e)
	case *semantic.EnumEntry:
		out = c.enumEntry(s, e)
	case semantic.Float32Value:
		out = c.float32Value(s, e)
	case semantic.Float64Value:
		out = c.float64Value(s, e)
	case *semantic.Global:
		out = c.global(s, e)
	case *semantic.Ignore:
		out = c.ignore(s, e)
	case semantic.Int16Value:
		out = c.int16Value(s, e)
	case semantic.Int32Value:
		out = c.int32Value(s, e)
	case semantic.Int64Value:
		out = c.int64Value(s, e)
	case semantic.Int8Value:
		out = c.int8Value(s, e)
	case *semantic.Length:
		out = c.length(s, e)
	case *semantic.Local:
		out = c.local(s, e)
	case *semantic.Make:
		out = c.make(s, e)
	case *semantic.MapContains:
		out = c.mapContains(s, e)
	case *semantic.MapIndex:
		out = c.mapIndex(s, e)
	case *semantic.Member:
		out = c.member(s, e)
	case *semantic.MessageValue:
		out = c.message(s, e)
	case semantic.Null:
		out = c.null(s, e)
	case *semantic.Observed:
		out = c.observed(s, e)
	case *semantic.Parameter:
		out = c.Parameter(s, e)
	case *semantic.PointerRange:
		out = c.pointerRange(s, e)
	case *semantic.Select:
		out = c.select_(s, e)
	case *semantic.SliceIndex:
		out = c.sliceIndex(s, e)
	case *semantic.SliceRange:
		out = c.sliceRange(s, e)
	case semantic.StringValue:
		out = c.stringValue(s, e)
	case semantic.Uint16Value:
		out = c.uint16Value(s, e)
	case semantic.Uint32Value:
		out = c.uint32Value(s, e)
	case semantic.Uint64Value:
		out = c.uint64Value(s, e)
	case semantic.Uint8Value:
		out = c.uint8Value(s, e)
	case *semantic.UnaryOp:
		out = c.unaryOp(s, e)
	case *semantic.Unknown:
		out = c.unknown(s, e)
	default:
		panic(fmt.Errorf("Unexpected expression type %T", e))
	}
	c.setCurrentExpression(s, old)
	return out
}

func (c *C) arrayIndex(s *S, e *semantic.ArrayIndex) *codegen.Value {
	return c.expressionAddr(s, e).Load()
}

func (c *C) arrayInitializer(s *S, e *semantic.ArrayInitializer) *codegen.Value {
	arr := s.Zero(c.T.Target(e.ExpressionType()))
	for i, e := range e.Values {
		arr = arr.Insert(i, c.expression(s, e))
	}
	return arr
}

func (c *C) binaryOp(s *S, e *semantic.BinaryOp) *codegen.Value {
	op, lhs := e.Operator, c.expression(s, e.LHS)
	switch op {
	case ast.OpBitShiftLeft:
		// RHS is always unsigned. JIT requires LHS and RHS type to be the same.
		rhs := c.doCast(s, e.LHS.ExpressionType(), e.RHS.ExpressionType(), c.expression(s, e.RHS))
		return c.doBinaryOp(s, op, lhs, rhs)
	case ast.OpBitShiftRight:
		// RHS is always unsigned. JIT requires LHS and RHS type to be the same.
		rhs := c.doCast(s, e.LHS.ExpressionType(), e.RHS.ExpressionType(), c.expression(s, e.RHS))
		return c.doBinaryOp(s, op, lhs, rhs)
	case ast.OpAnd:
		// Handle short-circuits.
		res := s.LocalInit("and-sc", s.Scalar(false))
		s.If(lhs, func(s *S) { res.Store(c.expression(s, e.RHS)) })
		return res.Load()
	case ast.OpOr:
		// Handle short-circuits.
		res := s.LocalInit("or-sc", s.Scalar(true))
		s.If(s.Not(lhs), func(s *S) { res.Store(c.expression(s, e.RHS)) })
		return res.Load()
	default:
		rhs := c.expression(s, e.RHS)
		return c.doBinaryOp(s, op, lhs, rhs)
	}
}

func (c *C) equal(s *S, lhs, rhs *codegen.Value) *codegen.Value {
	return c.doBinaryOp(s, "==", lhs, rhs)
}

func (c *C) doBinaryOp(s *S, op string, lhs, rhs *codegen.Value) *codegen.Value {
	if lhs.Type() == c.T.StrPtr {
		switch op {
		case ast.OpEQ, ast.OpGT, ast.OpLT, ast.OpGE, ast.OpLE, ast.OpNE:
			cmp := s.Call(c.callbacks.stringCompare, lhs, rhs)
			lhs, rhs = cmp, s.Zero(cmp.Type())
		}
	}

	switch op {
	case ast.OpEQ:
		return s.Equal(lhs, rhs)
	case ast.OpGT:
		return s.GreaterThan(lhs, rhs)
	case ast.OpLT:
		return s.LessThan(lhs, rhs)
	case ast.OpGE:
		return s.GreaterOrEqualTo(lhs, rhs)
	case ast.OpLE:
		return s.LessOrEqualTo(lhs, rhs)
	case ast.OpNE:
		return s.NotEqual(lhs, rhs)
	case ast.OpOr, ast.OpBitwiseOr:
		return s.Or(lhs, rhs)
	case ast.OpAnd, ast.OpBitwiseAnd:
		return s.And(lhs, rhs)
	case ast.OpPlus:
		if lhs.Type() == c.T.StrPtr {
			str := s.Call(c.callbacks.stringConcat, lhs, rhs)
			c.deferRelease(s, str, semantic.StringType)
			return str
		}
		return s.Add(lhs, rhs)
	case ast.OpMinus:
		return s.Sub(lhs, rhs)
	case ast.OpMultiply:
		return s.Mul(lhs, rhs)
	case ast.OpDivide:
		return s.Div(lhs, rhs)
	case ast.OpBitShiftLeft:
		return s.ShiftLeft(lhs, rhs)
	case ast.OpBitShiftRight:
		return s.ShiftRight(lhs, rhs)
	case ast.OpRange:
	case ast.OpIn:
	}
	fail("binary operator '%v' not implemented", op)
	return nil
}

func (c *C) bitTest(s *S, e *semantic.BitTest) *codegen.Value {
	bits := c.expression(s, e.Bits)
	bitfield := c.expression(s, e.Bitfield)
	mask := s.And(bits, bitfield)
	return s.NotEqual(mask, s.Zero(mask.Type()))
}

func (c *C) boolValue(s *S, e semantic.BoolValue) *codegen.Value {
	return s.Scalar(bool(e))
}

func (c *C) call(s *S, e *semantic.Call) *codegen.Value {
	tf := e.Target.Function
	args := make([]*codegen.Value, len(e.Arguments)+1)
	args[0] = s.Ctx
	for i, a := range e.Arguments {
		args[i+1] = c.expression(s, a).SetName(tf.FullParameters[i].Name())
	}
	f, ok := c.functions[tf]
	if !ok && tf.Subroutine {
		// Likely a subroutine calling another subrotine that hasn't been compiled yet.
		// Compile it now.
		c.subroutine(tf)
		f, ok = c.functions[tf]
	}
	if !ok {
		panic(fmt.Errorf("Couldn't resolve call target %v", tf.Name()))
	}

	res := s.Call(f, args...)

	if tf.Subroutine {
		// Subroutines return a <error, value> pair.
		// Check the error.
		err := res.Extract(retError)
		s.If(s.NotEqual(err, s.Scalar(ErrSuccess)), func(s *S) {
			retTy := c.returnType(c.currentFunc)
			s.Return(s.Zero(retTy).Insert(retError, err))
		})
		if tf.Return.Type == semantic.VoidType {
			return nil
		}
		// Return the value.
		res = res.Extract(retValue)
	}

	c.deferRelease(s, res, tf.Return.Type)

	return res
}

func (c *C) cast(s *S, e *semantic.Cast) *codegen.Value {
	dstTy := semantic.Underlying(e.Type)
	srcTy := semantic.Underlying(e.Object.ExpressionType())
	v := c.expression(s, e.Object)
	return c.doCast(s, dstTy, srcTy, v)
}

func (c *C) classInitializer(s *S, e *semantic.ClassInitializer) *codegen.Value {
	class := c.classInitializerNoRelease(s, e)
	c.deferRelease(s, class, e.Class)
	return class
}

func (c *C) classInitializerNoRelease(s *S, e *semantic.ClassInitializer) *codegen.Value {
	class := s.Undef(c.T.Target(e.ExpressionType()))
	for i, iv := range e.InitialValues() {
		if iv != nil {
			class = class.Insert(i, c.expression(s, iv))
		} else {
			class = class.Insert(i, c.initialValue(s, e.Class.Fields[i].Type))
		}
	}
	c.reference(s, class, e.Class) // references all referencable fields.
	return class
}

func (c *C) create(s *S, e *semantic.Create) *codegen.Value {
	refPtrTy := c.T.Target(e.Type).(codegen.Pointer)
	refTy := refPtrTy.Element
	ptr := c.Alloc(s, s.Scalar(uint64(1)), refTy)
	ptr.Index(0, RefRefCount).Store(s.Scalar(uint32(1)))
	ptr.Index(0, RefArena).Store(s.Arena)
	ptr.Index(0, RefValue).Store(c.classInitializerNoRelease(s, e.Initializer))
	c.deferRelease(s, ptr, e.Type)
	return ptr
}

func (c *C) clone(s *S, e *semantic.Clone) *codegen.Value {
	src := c.expression(s, e.Slice)
	srcSize := src.Extract(SliceSize)
	elSize := s.SizeOf(c.T.Storage(e.Type.To))
	dstCount := s.Div(srcSize, elSize)
	dstSize := s.Mul(dstCount, elSize)
	dst := c.MakeSlice(s, dstSize, dstCount)
	c.CopySlice(s, dst, src)
	c.deferRelease(s, dst, e.Type)
	return dst
}

func (c *C) enumEntry(s *S, e *semantic.EnumEntry) *codegen.Value {
	return s.Scalar(e.Value)
}

func (c *C) float32Value(s *S, e semantic.Float32Value) *codegen.Value {
	return s.Scalar(float32(e))
}

func (c *C) float64Value(s *S, e semantic.Float64Value) *codegen.Value {
	return s.Scalar(float64(e))
}

func (c *C) global(s *S, e *semantic.Global) *codegen.Value {
	if e.Name() == semantic.BuiltinThreadGlobal.Name() {
		return s.Scalar(uint64(0)) // TODO
	}
	return s.Globals.Index(0, e.Name()).Load()
}

func (c *C) ignore(s *S, e *semantic.Ignore) *codegen.Value {
	panic("Unreachable")
}

func (c *C) int8Value(s *S, e semantic.Int8Value) *codegen.Value {
	return s.Scalar(int8(e))
}

func (c *C) int16Value(s *S, e semantic.Int16Value) *codegen.Value {
	return s.Scalar(int16(e))
}

func (c *C) int32Value(s *S, e semantic.Int32Value) *codegen.Value {
	return s.Scalar(int32(e))
}

func (c *C) int64Value(s *S, e semantic.Int64Value) *codegen.Value {
	return s.Scalar(int64(e))
}

func (c *C) length(s *S, e *semantic.Length) *codegen.Value {
	o := c.expression(s, e.Object)
	var l *codegen.Value
	switch ty := semantic.Underlying(e.Object.ExpressionType()).(type) {
	case *semantic.Slice:
		l = o.Extract(SliceCount)
	case *semantic.Map:
		l = o.Index(0, MapCount).Load()
	case *semantic.Builtin:
		switch ty {
		case semantic.StringType:
			l = o.Index(0, StringLength).Load()
		}
	}
	if l == nil {
		fail("Unhandled length expression type %v", e.Object.ExpressionType().Name())
	}
	return c.doCast(s, e.Type, semantic.Uint32Type, l)
}

func (c *C) local(s *S, e *semantic.Local) *codegen.Value {
	l, ok := s.locals[e]
	if !ok {
		locals := make([]string, 0, len(s.locals))
		for l := range s.locals {
			locals = append(locals, fmt.Sprintf(" • %v", l.Name()))
		}
		fail("Couldn't locate local '%v'. Have locals:\n%v", e.Name(), strings.Join(locals, "\n"))
	}
	return l.Load()
}

func (c *C) make(s *S, e *semantic.Make) *codegen.Value {
	elTy := c.T.Storage(e.Type.To)
	count := c.expression(s, e.Size).Cast(c.T.Uint64)
	size := s.Mul(count, s.SizeOf(elTy))
	slice := c.MakeSlice(s, size, count)
	c.deferRelease(s, slice, e.Type)
	return slice
}

func (c *C) mapContains(s *S, e *semantic.MapContains) *codegen.Value {
	m := c.expression(s, e.Map)
	k := c.expression(s, e.Key)
	return s.Call(c.T.Maps[e.Type].Contains, m, k).SetName("map_contains")
}

func (c *C) mapIndex(s *S, e *semantic.MapIndex) *codegen.Value {
	m := c.expression(s, e.Map)
	k := c.expression(s, e.Index)
	res := s.Call(c.T.Maps[e.Type].Lookup, m, k).SetName("map_lookup")
	c.deferRelease(s, res, e.Type.ValueType)
	return res
}

func (c *C) member(s *S, e *semantic.Member) *codegen.Value {
	obj := c.expression(s, e.Object)
	switch ty := semantic.Underlying(e.Object.ExpressionType()).(type) {
	case *semantic.Class:
		return obj.Extract(e.Field.Name())
	case *semantic.Reference:
		return obj.Index(0, RefValue, e.Field.Name()).Load()
	default:
		fail("Unexpected type for member: '%v'", ty)
		return nil
	}
}

func (c *C) message(s *S, e *semantic.MessageValue) *codegen.Value {
	return s.Zero(c.T.Target(e.ExpressionType())).SetName("TODO:message") // TODO
}

func (c *C) null(s *S, e semantic.Null) *codegen.Value {
	return s.Zero(c.T.Target(e.Type))
}

func (c *C) observed(s *S, e *semantic.Observed) *codegen.Value {
	return c.Parameter(s, e.Parameter)
}

// Parameter returns the loaded parameter value, failing if the parameter cannot
// be found.
func (c *C) Parameter(s *S, e *semantic.Parameter) *codegen.Value {
	p, ok := s.Parameters[e]
	if !ok {
		params := make([]string, 0, len(s.Parameters))
		for p := range s.Parameters {
			params = append(params, fmt.Sprintf(" • %v", p.Name()))
		}
		c.Fail("Couldn't locate parameter '%v'. Have parameters:\n%v",
			e.Name(), strings.Join(params, "\n"))
	}
	return p
}

func (c *C) pointerRange(s *S, e *semantic.PointerRange) *codegen.Value {
	p := c.expression(s, e.Pointer)
	elTy := c.T.Storage(e.Type.To)
	start := c.expression(s, e.Range.LHS).Cast(c.T.Uint64).SetName("start")
	end := c.expression(s, e.Range.RHS).Cast(c.T.Uint64).SetName("end")
	offset := s.Mul(start, s.SizeOf(elTy)).Cast(c.T.Uint64).SetName("offset")
	count := s.Sub(end, start).SetName("count")
	size := s.Mul(count, s.SizeOf(elTy)).Cast(c.T.Uint64).SetName("size")
	slicePtr := s.Local("slicePtr", c.T.Sli)
	s.Call(c.callbacks.pointerToSlice, s.Ctx, p, offset, size, count, slicePtr)
	slice := slicePtr.Load()
	c.deferRelease(s, slice, e.Type)
	return slice
}

func (c *C) select_(s *S, e *semantic.Select) *codegen.Value {
	val := c.expression(s, e.Value)

	cases := make([]SwitchCase, len(e.Choices))
	res := s.Local("select_result", c.T.Target(e.Type))
	for i, choice := range e.Choices {
		i, choice := i, choice
		cases[i] = SwitchCase{
			Conditions: func(s *S) []*codegen.Value {
				conds := make([]*codegen.Value, len(choice.Conditions))
				for i, cond := range choice.Conditions {
					conds[i] = c.equal(s, val, c.expression(s, cond))
				}
				return conds
			},
			Block: func(s *S) {
				val := c.expression(s, choice.Expression)
				c.reference(s, val, e.Type)
				res.Store(val)
			},
		}
	}

	var def func(s *S)
	if e.Default != nil {
		def = func(s *S) {
			val := c.expression(s, e.Default)
			c.reference(s, val, e.Type)
			res.Store(val)
		}
	}

	s.Switch(cases, def)

	out := res.Load()
	c.deferRelease(s, out, e.Type)
	return out
}

func (c *C) sliceIndex(s *S, e *semantic.SliceIndex) *codegen.Value {
	index := c.expression(s, e.Index).Cast(c.T.Uint64).SetName("index")
	slice := c.expression(s, e.Slice)

	read := func(elType codegen.Type) *codegen.Value {
		base := slice.Extract(SliceBase).Cast(c.T.Pointer(elType))
		return base.Index(index).Load()
	}

	elTy := e.Type.To
	targetTy := c.T.Target(e.Type.To)
	storageTy := c.T.Storage(e.Type.To)
	if targetTy == storageTy {
		return read(targetTy)
	}
	return c.castStorageToTarget(s, elTy, read(storageTy))
}

func (c *C) sliceRange(s *S, e *semantic.SliceRange) *codegen.Value {
	slice := c.expression(s, e.Slice)
	elTy := c.T.Storage(e.Type.To)
	elPtrTy := c.T.Pointer(elTy)
	base := slice.Extract(SliceBase).Cast(elPtrTy) // T*
	from := c.expression(s, e.Range.LHS).SetName("slice_from")
	to := c.expression(s, e.Range.RHS).SetName("slice_to")
	start := base.Index(from).SetName("slice_start")                                  // T*
	end := base.Index(to).SetName("slice_end")                                        // T*
	size := s.Sub(end.Cast(c.T.Uint64), start.Cast(c.T.Uint64)).SetName("slice_size") // u64

	slice = slice.Insert(SliceCount, s.Sub(to, from))
	slice = slice.Insert(SliceSize, size)
	slice = slice.Insert(SliceBase, start.Cast(c.T.Uint8Ptr))
	// TODO: Check sub-slice is within original slice bounds.
	return slice
}

func (c *C) stringValue(s *S, e semantic.StringValue) *codegen.Value {
	str := c.MakeString(s, s.Scalar(uint64(len(e))), s.GlobalString(string(e)))
	c.deferRelease(s, str, semantic.StringType)
	return str
}

func (c *C) uint8Value(s *S, e semantic.Uint8Value) *codegen.Value {
	return s.Scalar(uint8(e))
}

func (c *C) uint16Value(s *S, e semantic.Uint16Value) *codegen.Value {
	return s.Scalar(uint16(e))
}

func (c *C) uint32Value(s *S, e semantic.Uint32Value) *codegen.Value {
	return s.Scalar(uint32(e))
}

func (c *C) uint64Value(s *S, e semantic.Uint64Value) *codegen.Value {
	return s.Scalar(uint64(e))
}

func (c *C) unaryOp(s *S, e *semantic.UnaryOp) *codegen.Value {
	switch e.Operator {
	case ast.OpNot:
		return s.Not(c.expression(s, e.Expression))
	}
	fail("unary operator '%v' not implemented", e.Operator)
	return nil
}

func (c *C) unknown(s *S, e *semantic.Unknown) *codegen.Value {
	return c.expression(s, e.Inferred)
}
