package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type LinkInstruction struct {
	Source string
	Target string
	Force  bool
}

func (instruction *LinkInstruction) String() string {
	return fmt.Sprintf("%s%s%s%s%s%s%s%s%s", SourceColor, instruction.Source, DefaultColor, deliminer, TargetColor, instruction.Target, DefaultColor, deliminer, strconv.FormatBool(instruction.Force))
}

func ParseInstruction(line string) (LinkInstruction, error) {
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

func (instruction *LinkInstruction) RunSync() error {
	if !FileExists(instruction.Source) {
		return fmt.Errorf("instruction source %s%s%s does not exist", SourceColor, instruction.Source, DefaultColor)
	}

	if AreSame(instruction.Source, instruction.Target) {
		log.Printf("Source %s%s%s and target %s%s%s are the same, %snothing to do...%s", SourceColor, instruction.Source, DefaultColor, TargetColor, instruction.Target, DefaultColor, PathColor, DefaultColor)
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
	log.Printf("Created symlink between %s%s%s and %s%s%s", SourceColor, instruction.Source, DefaultColor, TargetColor, instruction.Target, DefaultColor)

	return nil
}

func (instruction *LinkInstruction) RunAsync(status chan (error)) {
	if !FileExists(instruction.Source) {
		status <- fmt.Errorf("instruction source %s%s%s does not exist", SourceColor, instruction.Source, DefaultColor)
	}

	if AreSame(instruction.Source, instruction.Target) {
		status <- fmt.Errorf("source %s%s%s and target %s%s%s are the same, %snothing to do...%s", SourceColor, instruction.Source, DefaultColor, TargetColor, instruction.Target, DefaultColor, PathColor, DefaultColor)
	}

	if FileExists(instruction.Target) {
		if instruction.Force {
			isSymlink, err := IsSymlink(instruction.Target)
			if err != nil {
				status <- fmt.Errorf("could not determine whether %s%s%s is a sym link or not, stopping; err: %s%+v%s", TargetColor, instruction.Target, DefaultColor, ErrorColor, err, DefaultColor)
			}

			if isSymlink {
				log.Printf("Removing symlink at %s%s%s", TargetColor, instruction.Target, DefaultColor)
				err = os.Remove(instruction.Target)
				if err != nil {
					status <- fmt.Errorf("failed deleting %s%s%s due to %s%+v%s", TargetColor, instruction.Target, DefaultColor, ErrorColor, err, DefaultColor)
				}
			} else {
				status <- fmt.Errorf("refusing to delte actual (non symlink) file %s%s%s", TargetColor, instruction.Target, DefaultColor)
			}
		} else {
			status <- fmt.Errorf("target %s%s%s exists - handle manually or set the 'forced' flag (3rd field)", TargetColor, instruction.Target, DefaultColor)
		}
	}

	err := os.Symlink(instruction.Source, instruction.Target)
	if err != nil {
		status <- fmt.Errorf("failed creating symlink between %s%s%s and %s%s%s with error %s%+v%s", SourceColor, instruction.Source, DefaultColor, TargetColor, instruction.Target, DefaultColor, ErrorColor, err, DefaultColor)
	}
	log.Printf("Created symlink between %s%s%s and %s%s%s", SourceColor, instruction.Source, DefaultColor, TargetColor, instruction.Target, DefaultColor)

	status <- nil
}