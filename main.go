package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const deliminer = ","
const (
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
)
const SourceColor = Magenta
const TargetColor = Yellow
const ErrorColor = Red
const DefaultColor = White

type LinkInstruction struct {
	Source string
	Target string
	Force  bool
}

func InstructionToString(instruction LinkInstruction) string {
	return fmt.Sprintf("%s%s%s%s%s%s%s%s%s", SourceColor, instruction.Source, DefaultColor, deliminer, TargetColor, instruction.Target, DefaultColor, deliminer, strconv.FormatBool(instruction.Force))
}

func main() {
	// Format:
	// source|target|force?
	log.SetFlags(log.Lmicroseconds)
	var inputs []string
	// os.Chdir("C:/Users/Administrator/Seafile/Last-Epoch-Backup")

	if len(os.Args) > 1 {
		inputs = append(inputs, os.Args[1:]...)
	} else {
		info, err := os.Stdin.Stat()
		if err != nil {
			log.Fatalf("Failed to stat stdin: %s%+v%s", ErrorColor, err, DefaultColor)
		}
		if info.Mode()&os.ModeNamedPipe != 0 {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				inputs = append(inputs, scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				log.Fatalf("Error reading from stdin: %s%+v%s", ErrorColor, err, DefaultColor)
			}
		}
	}

	if len(inputs) == 0 {
		log.Printf("No input provided")
		log.Printf("Supply input as either arguments (source,target,force?)")
		log.Printf("or via stdin (cat <file> | %s)", os.Args[0])
		os.Exit(1)
	}

	for _, line := range inputs {
		instruction, err := parseLine(line)
		if err != nil {
			log.Printf("Error parsing line: %s'%s'%s, error: %s%+v%s", SourceColor, line, DefaultColor, ErrorColor, err, DefaultColor)
			continue
		}
		log.Printf("Processing: %s", InstructionToString(instruction))
		err = processInstruction(instruction)
		if err != nil {
			log.Printf("Failed parsing instruction %s%s%s due to %s%+v%s", SourceColor, InstructionToString(instruction), DefaultColor, ErrorColor, err, DefaultColor)
			continue
		}
	}
}
func parseLine(line string) (LinkInstruction, error) {
	parts := strings.Split(line, deliminer)
	instruction := LinkInstruction{}

	if len(parts) < 2 {
		return instruction, fmt.Errorf("invalid format - not enough parameters (must have at least source and target)")
	}

	instruction.Source = parts[0]
	instruction.Target = parts[1]
	instruction.Force = false
	if len(parts) > 2 {
		res, _ := regexp.MatchString("^(?i)T|TRUE$", parts[2])
		instruction.Force = res
	}

	instruction.Source, _ = ConvertHome(instruction.Source)
	instruction.Target, _ = ConvertHome(instruction.Target)

	instruction.Source = NormalizePath(instruction.Source)
	instruction.Target = NormalizePath(instruction.Target)

	return instruction, nil
}

func processInstruction(instruction LinkInstruction) error {
	if !FileExists(instruction.Source) {
		return fmt.Errorf("instruction source %s%s%s does not exist", SourceColor, instruction.Source, DefaultColor)
	}

	if AreSame(instruction.Source, instruction.Target) {
		log.Printf("Source %s%s%s and target %s%s%s are the same, nothing to do...", SourceColor, instruction.Source, DefaultColor, TargetColor, instruction.Target, DefaultColor)
		return nil
	}

	if FileExists(instruction.Target) {
		if instruction.Force {
			isSymlink, err := IsSymlink(instruction.Target)
			if err != nil {
				return fmt.Errorf("could not determine whether %s%s%s is a sym link or not, stopping; err: %s%+v%s", TargetColor, instruction.Target, DefaultColor, ErrorColor, err, DefaultColor)
			}

			if isSymlink {
				log.Printf("Removing symlink at %s%s%s", TargetColor, instruction.Target, DefaultColor)
				err = os.Remove(instruction.Target)
				if err != nil {
					return fmt.Errorf("failed deleting %s%s%s due to %s%+v%s", TargetColor, instruction.Target, DefaultColor, ErrorColor, err, DefaultColor)
				}
			} else {
				return fmt.Errorf("refusing to delte actual (non symlink) file %s%s%s", TargetColor, instruction.Target, DefaultColor)
			}
		} else {
			return fmt.Errorf("target %s%s%s exists - handle manually or set the 'forced' flag (3rd field)", TargetColor, instruction.Target, DefaultColor)
		}
	}

	err := os.Symlink(instruction.Source, instruction.Target)
	if err != nil {
		return fmt.Errorf("failed creating symlink between %s%s%s and %s%s%s with error %s%+v%s", SourceColor, instruction.Source, DefaultColor, TargetColor, instruction.Target, DefaultColor, ErrorColor, err, DefaultColor)
	}

	return nil
}
