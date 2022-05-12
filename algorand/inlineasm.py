#!/usr/bin/python3
from pyteal import *


class CustomOp():
    def __init__(self, opcode):
        self.opcode = opcode
        self.mode = Mode.Signature | Mode.Application
        self.min_version = 2


    def __str__(self) -> str:
        return self.opcode


class InlineAssembly(LeafExpr):
    def __init__(self, opcode: str, *args: "Expr", type: TealType = TealType.none) -> None:
        super().__init__()
        opcode_with_args = opcode.split(" ")
        self.op = CustomOp(opcode_with_args[0])
        self.type = type
        self.opcode_args = opcode_with_args[1:]
        self.args = args


    def __teal__(self, options: "CompileOptions"):
        op = TealOp(self, self.op, *self.opcode_args)
        return TealBlock.FromOp(options, op, *self.args[::1])


    def __str__(self):
        return "(InlineAssembly: {})".format(self.opcode)


    def type_of(self):
        return self.type
