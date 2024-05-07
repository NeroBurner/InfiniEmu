package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrExit            = errors.New("exit")
	ErrInvalidArgument = errors.New("invalid argument")
)

const digits = "0123456789"

type Command func(mod, arg string) error

var Commands = map[string]Command{
	"next":     CommandNextPreviousFrame(1),
	"previous": CommandNextPreviousFrame(-1),
	"frame":    CommandFrame,
	"view":     CommandView,
	"eval":     CommandEval,
	"exit":     func(string, string) error { return ErrExit },
}

func printInt(v uint32, modifier string) {
	if strings.ContainsRune(modifier, 'd') {
		fmt.Printf("%d\n", v)
	} else if strings.ContainsRune(modifier, 's') {
		fmt.Printf("%d\n", int32(v))
	} else if strings.ContainsRune(modifier, 'b') {
		fmt.Printf("0b%032b\n", int32(v))
	} else {
		fmt.Printf("0x%08x\n", v)
	}
}

func FindCommand(name string) (Command, error) {
	cmd, ok := Commands[name]
	if ok {
		return cmd, nil
	}

	matches := []string{}

	for k := range Commands {
		if strings.HasPrefix(k, name) {
			matches = append(matches, k)
		}
	}

	if len(matches) == 0 {
		return nil, errors.New("unknown command")
	}
	if len(matches) == 1 {
		return Commands[matches[0]], nil
	}

	return nil, fmt.Errorf("ambiguous command: %s", strings.Join(matches, ", "))
}

func CommandNextPreviousFrame(dir int) Command {
	return func(_, arg string) error {
		offset := dir

		if arg != "" {
			n, err := parseInt(arg)
			if err != nil {
				return ErrInvalidArgument
			}

			offset *= n
		}

		if offset > 0 && frameIndex+offset >= len(frames) {
			frameIndex = len(frames) - 1
		} else if offset < 0 && frameIndex+offset < 0 {
			frameIndex = 0
		} else {
			frameIndex += offset
		}

		return nil
	}
}

func CommandFrame(_, arg string) error {
	if arg == "" {
		fmt.Printf("On frame %d out of %d\n", frameIndex, len(frames))
	} else if arg == "end" {
		frameIndex = len(frames) - 1
	} else {
		n, err := parseInt(arg)
		if err != nil || n < 0 || n >= len(frames) {
			return errors.New("invalid or out of range frame index")
		}

		frameIndex = n
	}

	return nil
}

func CommandView(modifier, arg string) error {
	currentFrame := frames[frameIndex]

	if arg == "" {
		for i := RUNLOG_REG_R0; i <= RUNLOG_REG_MAX; i++ {
			fmt.Printf("%s\t0x%08x\n", i.String(), currentFrame.Registers[i])
		}
	} else {
		baseAddr, err := parseInt(arg)
		if err != nil {
			return errors.New("invalid address")
		}

		count := 1
		if modifier != "" {
			firstNum := strings.LastIndexAny(modifier, digits)
			lastNum := strings.LastIndexAny(modifier, digits)

			if firstNum == 0 {
				count, err = strconv.Atoi(modifier[firstNum : lastNum+1])
				if err != nil {
					return errors.New("invalid count")
				}
			} else if firstNum != -1 {
				return errors.New("invalid modifier")
			}
		}

		for i := uint32(0); i < uint32(count); i++ {
			addr := uint32(baseAddr) + i*4

			val, err := frames[:frameIndex+1].ReadMemoryAt(addr)
			if err != nil {
				return err
			}

			fmt.Printf("0x%08x: ", addr)
			printInt(val, modifier)
		}
	}

	return nil
}

func CommandEval(modifier, arg string) error {
	expr, err := ParseExpression(arg)
	if err != nil {
		return err
	}

	size := 4
	count := 1

	if modifier != "" {
		firstNum := strings.LastIndexAny(modifier, digits)
		lastNum := strings.LastIndexAny(modifier, digits)

		if firstNum == 0 {
			count, err = strconv.Atoi(modifier[firstNum : lastNum+1])
			if err != nil {
				return errors.New("invalid count")
			}
		} else if firstNum != -1 {
			return errors.New("invalid modifier")
		}

		if strings.ContainsRune(modifier, 'b') {
			size = 1
		} else if strings.ContainsRune(modifier, 'h') {
			size = 2
		}
	}

	for i := 0; i < count; i++ {
		val, err := expr.Evaluate(ExpressionContext{
			Frames: frames[:frameIndex+1],
			Offset: uint32(i * size),
		})
		if err != nil {
			return fmt.Errorf("at %d: %w", i, err)
		}

		printInt(val, modifier)
	}

	return nil
}
