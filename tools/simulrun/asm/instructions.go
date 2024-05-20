package asm

import (
	"fmt"
	"math/rand"
	"strings"
)

type Register uint8

const (
	RegisterR0 Register = iota
	RegisterR1
	RegisterR2
	RegisterR3
	RegisterR4
	RegisterR5
	RegisterR6
	RegisterR7
	RegisterR8
	RegisterR9
	RegisterR10
	RegisterR11
	RegisterR12
	RegisterSP
	RegisterLR
	RegisterPC
	RegisterXPSR
)

func (r Register) String() string {
	if r <= 12 {
		return fmt.Sprintf("r%d", r)
	}

	switch r {
	case RegisterSP:
		return "sp"
	case RegisterLR:
		return "lr"
	case RegisterPC:
		return "pc"
	default:
		return "unknown"
	}
}

func (r Register) withMax(max uint32) FuzzedRegister {
	return FuzzedRegister{
		Register: r,
		Maximum:  max,
	}
}

type FuzzedRegister struct {
	Register Register
	Minimum  uint32
	Maximum  uint32
}

type XPSR uint32

func (x XPSR) N() bool {
	return x&(1<<31) != 0
}

func (x XPSR) Z() bool {
	return x&(1<<30) != 0
}

func (x XPSR) C() bool {
	return x&(1<<29) != 0
}

func (x XPSR) V() bool {
	return x&(1<<28) != 0
}

type ShiftType uint8

const (
	ShiftLSL ShiftType = iota
	ShiftLSR
	ShiftASR
	ShiftROR
	ShiftRRX
)

func (s ShiftType) String() string {
	switch s {
	case ShiftLSL:
		return "lsl"
	case ShiftLSR:
		return "lsr"
	case ShiftASR:
		return "asr"
	case ShiftROR:
		return "ror"
	case ShiftRRX:
		return "rrx"
	default:
		panic("invalid shift type")
	}
}

type RegisterShift struct {
	Type   ShiftType
	Amount uint
}

func (r RegisterShift) IsEmpty() bool {
	return r.Type != ShiftRRX && r.Amount == 0
}

func (r RegisterShift) String() string {
	if r.Type == ShiftRRX {
		return r.Type.String()
	}
	if r.Amount == 0 {
		return ""
	}

	return fmt.Sprintf("%s #%d", r.Type, r.Amount)
}

type Instruction struct {
	Name     string
	Flags    InstructionFlags
	Operands []any
}

func (i Instruction) String() string {
	name := i.Name

	if i.Flags.Has(FlagUpdateFlags) {
		name += "s"
	}
	if i.Flags.Has(FlagWide) {
		name += ".w"
	}

	opStrings := make([]string, 0, len(i.Operands))

	for _, op := range i.Operands {
		switch op := op.(type) {
		case Register:
			opStrings = append(opStrings, op.String())
		case FuzzedRegister:
			opStrings = append(opStrings, op.Register.String())
		case uint32, int32:
			opStrings = append(opStrings, fmt.Sprintf("#%d", op))
		case RegisterShift:
			if !op.IsEmpty() {
				opStrings = append(opStrings, op.String())
			}
		default:
			panic("invalid operand type")
		}
	}

	return fmt.Sprintf("%s %s", name, strings.Join(opStrings, ", "))
}

type InstructionFlags uint32

const (
	FlagNone        InstructionFlags = 0
	FlagUpdateFlags InstructionFlags = 1 << iota
	FlagMaybeUpdateFlags
	FlagWide
)

func (f InstructionFlags) Has(flag InstructionFlags) bool {
	return f&flag != 0
}

type RandASM struct {
	*rand.Rand
}

func (r RandASM) maybe() bool {
	return r.Int63()%2 == 0
}

func (r RandASM) MaybeNegative(n uint32) int32 {
	if r.maybe() {
		return -int32(n)
	}

	return int32(n)
}

func (r RandASM) RandIntBits(n int) uint32 {
	return uint32(r.Int63()) & (1<<n - 1)
}

func (r RandASM) RandRegister() Register {
	return Register(r.Intn(16))
}

func (r RandASM) RandLowRegister() Register {
	return Register(r.Intn(13))
}

func (r RandASM) RandRegisterBits(n int) Register {
	return Register(r.Intn(n))
}

func (r RandASM) RandThumbImm() uint32 {
	imm8 := r.RandIntBits(8)

	if r.maybe() {
		// Simple value

		switch r.Int63() % 4 {
		case 0:
			return imm8
		case 1:
			return imm8 | (imm8 << 16)
		case 2:
			return (imm8 << 8) | (imm8 << 24)
		case 3:
			return imm8 | (imm8 << 8) | (imm8 << 16) | (imm8 << 24)
		default:
			panic("unreachable")
		}
	} else {
		// Rotated value

		lsl := (r.Int63() % 24) + 1
		imm8 |= 1 << 7

		return imm8 << lsl
	}
}

func (r RandASM) RandUpdateFlags(insName string, ops string, a ...any) string {
	if r.maybe() {
		insName += "s"
	}

	return fmt.Sprintf("%s "+ops, append([]any{insName}, a...)...)
}

func (r RandASM) RandShift() RegisterShift {
	if r.maybe() {
		return RegisterShift{
			Type:   ShiftType(r.Intn(5)),
			Amount: uint(r.Intn(32)),
		}
	}

	return RegisterShift{}
}

func (r RandASM) inst(name string, flags InstructionFlags, ops ...any) Instruction {
	if flags.Has(FlagMaybeUpdateFlags) && r.maybe() {
		flags |= FlagUpdateFlags
	}

	return Instruction{
		Name:     name,
		Flags:    flags,
		Operands: ops,
	}
}

type Generator func(r RandASM) Instruction

var Instructions = []Generator{
	// ADC (immediate)
	func(r RandASM) Instruction {
		return r.inst("adc", FlagMaybeUpdateFlags, r.RandLowRegister(), r.RandLowRegister(), r.RandThumbImm())
	},
	// ADC (register) T1
	func(r RandASM) Instruction {
		return r.inst("adc", FlagMaybeUpdateFlags, r.RandLowRegister(), r.RandLowRegister())
	},
	// ADC (register) T2
	func(r RandASM) Instruction {
		return r.inst("adc", FlagMaybeUpdateFlags, r.RandLowRegister(), r.RandLowRegister(), r.RandLowRegister(), r.RandShift())
	},

	// ADD (immediate) T1
	func(r RandASM) Instruction {
		return r.inst("add", FlagNone, r.RandLowRegister(), r.RandLowRegister(), r.RandIntBits(3))
	},
	// ADD (immediate) T2
	func(r RandASM) Instruction {
		return r.inst("add", FlagNone, r.RandLowRegister(), r.RandLowRegister(), r.RandIntBits(8))
	},
	// ADD (immediate) T3
	func(r RandASM) Instruction {
		return r.inst("add", FlagMaybeUpdateFlags|FlagWide, r.RandLowRegister(), r.RandLowRegister(), r.RandThumbImm())
	},
	// ADD (immediate) T4
	func(r RandASM) Instruction {
		return r.inst("add", FlagWide, r.RandLowRegister(), r.RandLowRegister(), r.RandIntBits(12))
	},

	// ADD (register) T1
	func(r RandASM) Instruction {
		return r.inst("add", FlagNone, r.RandRegisterBits(3), r.RandRegisterBits(3), r.RandRegisterBits(3))
	},
	// ADD (register) T2
	func(r RandASM) Instruction {
		return r.inst("add", FlagNone, r.RandLowRegister(), r.RandLowRegister())
	},
	// ADD (register) T3
	func(r RandASM) Instruction {
		return r.inst("add", FlagWide, r.RandLowRegister(), r.RandLowRegister(), r.RandLowRegister(), r.RandShift())
	},

	// ADD (SP plus immediate) T1
	func(r RandASM) Instruction {
		return r.inst("add", FlagNone, r.RandRegisterBits(3), RegisterSP, r.RandIntBits(8))
	},
	// ADD (SP plus immediate) T2
	func(r RandASM) Instruction {
		return r.inst("add", FlagNone, RegisterSP, RegisterSP, r.RandIntBits(7)<<2)
	},
	// ADD (SP plus immediate) T3
	func(r RandASM) Instruction {
		return r.inst("add", FlagMaybeUpdateFlags|FlagWide, r.RandLowRegister(), RegisterSP, r.RandThumbImm())
	},
	// ADD (SP plus immediate) T4
	func(r RandASM) Instruction {
		return r.inst("add", FlagWide, r.RandLowRegister(), RegisterSP, r.RandIntBits(12))
	},

	// ADD (SP plus register) T1
	func(r RandASM) Instruction {
		reg := r.RandLowRegister()
		return r.inst("add", FlagNone, reg, RegisterSP, reg)
	},
	// ADD (SP plus register) T2
	func(r RandASM) Instruction {
		return r.inst("add", FlagNone, RegisterSP, r.RandLowRegister())
	},
	// ADD (SP plus register) T3
	func(r RandASM) Instruction {
		return r.inst("add", FlagMaybeUpdateFlags|FlagWide, r.RandLowRegister(), RegisterSP, r.RandLowRegister(), r.RandShift())
	},

	// These seem to emit SUB instructions instead of ADR
	// ADR T1
	func(r RandASM) Instruction {
		return r.inst("adr", FlagNone, r.RandLowRegister(), r.RandIntBits(8)<<2)
	},
	// ADR T2, T3
	func(r RandASM) Instruction {
		return r.inst("adr", FlagWide, r.RandLowRegister(), r.MaybeNegative(r.RandIntBits(12)))
	},

	// AND (immediate) T1
	func(r RandASM) Instruction {
		return r.inst("and", FlagMaybeUpdateFlags, r.RandLowRegister(), r.RandLowRegister(), r.RandThumbImm())
	},

	// AND (register) T1
	func(r RandASM) Instruction {
		return r.inst("and", FlagMaybeUpdateFlags, r.RandLowRegister(), r.RandLowRegister())
	},
	// AND (register) T2
	func(r RandASM) Instruction {
		return r.inst("and", FlagMaybeUpdateFlags|FlagWide, r.RandLowRegister(), r.RandLowRegister(), r.RandLowRegister(), r.RandShift())
	},

	// ASR (immediate) T1
	func(r RandASM) Instruction {
		return r.inst("asr", FlagNone, r.RandRegisterBits(3), r.RandRegisterBits(3), r.RandIntBits(5))
	},
	// ASR (immediate) T2
	func(r RandASM) Instruction {
		return r.inst("asr", FlagMaybeUpdateFlags|FlagWide, r.RandLowRegister(), r.RandLowRegister(), r.RandIntBits(5))
	},

	// ASR (register) T1
	func(r RandASM) Instruction {
		return r.inst("asr", FlagNone, r.RandRegisterBits(3), r.RandRegisterBits(3))
	},
	// ASR (register) T2
	func(r RandASM) Instruction {
		return r.inst("asr", FlagMaybeUpdateFlags|FlagWide, r.RandLowRegister(), r.RandLowRegister(), r.RandLowRegister())
	},
}
