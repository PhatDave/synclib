package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"regexp"
	"sync"
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
const PathColor = Green

var DirRegex, _ = regexp.Compile(`^(.+?)[/\\]sync$`)
var programName = os.Args[0]

func main() {
	// Format:
	// source,target,force?
	log.SetFlags(log.Lmicroseconds)

	recurse := flag.String("r", "", "recurse into directories")
	file := flag.String("f", "", "file to read instructions from")
	flag.Parse()

	log.Printf("Recurse: %s", *recurse)
	log.Printf("File: %s", *file)

	var instructions []LinkInstruction
	if *recurse != "" {
		instructions, _ = ReadFromFilesRecursively(*recurse)
	} else if *file != "" {
		instructions, _ = ReadFromFile(*file)
	} else if len(os.Args) > 1 {
		instructions, _ = ReadFromArgs()
	} else {
		instructions, _ = ReadFromStdin()
	}

	if len(instructions) == 0 {
		log.Printf("No input provided")
		log.Printf("Provide input as: ")
		log.Printf("Arguments - %s <source>,<target>,<force?>", programName)
		log.Printf("File - %s -f <file>", programName)
		log.Printf("Folder (finding sync files in folder recursively) - %s -r <folder>", programName)
		log.Printf("stdin - (cat <file> | %s)", programName)
		os.Exit(1)
	}
	var wg sync.WaitGroup
	for _, instruction := range instructions {
		log.Printf("Processing: %s", instruction.String())
		status := make(chan error)
		go instruction.RunAsync(status)
		wg.Add(1)
		err := <-status
		if err != nil {
			log.Printf("Failed parsing instruction %s%s%s due to %s%+v%s", SourceColor, instruction.String(), DefaultColor, ErrorColor, err, DefaultColor)
		}
		wg.Done()
	}
	wg.Wait()
}

func ReadFromFilesRecursively(input string) ([]LinkInstruction, error) {
	input = NormalizePath(input)
	log.Printf("Reading input from files recursively starting in %s%s%s", PathColor, input, DefaultColor)
	var instructions []LinkInstruction

	files, err := GetSyncFilesRecursively(input)
	if err != nil {
		log.Fatalf("Failed to get sync files recursively: %s%+v%s", ErrorColor, err, DefaultColor)
	}

	for _, file := range files {
		file = NormalizePath(file)

		// This "has" to be done because instructions are resolved in relation to cwd
		fileDir := DirRegex.FindStringSubmatch(file)
		if fileDir == nil {
			log.Printf("Failed to extract directory from %s%s%s", SourceColor, file, DefaultColor)
			continue
		}
		log.Printf("Changing directory to %s%s%s (for %s%s%s)", PathColor, fileDir[1], DefaultColor, PathColor, file, DefaultColor)
		err := os.Chdir(fileDir[1])
		if err != nil {
			log.Printf("Failed to change directory to %s%s%s: %s%+v%s", SourceColor, fileDir[1], DefaultColor, ErrorColor, err, DefaultColor)
			continue
		}

		fileInstructions, _ := ReadFromFile(file)
		instructions = append(instructions, fileInstructions...)
	}
	return instructions, nil
}
func ReadFromFile(input string) ([]LinkInstruction, error) {
	input = NormalizePath(input)
	log.Printf("Reading input from file: %s%s%s", PathColor, input, DefaultColor)
	var instructions []LinkInstruction
	file, err := os.Open(input)
	if err != nil {
		log.Fatalf("Failed to open file %s%s%s: %s%+v%s", SourceColor, input, DefaultColor, ErrorColor, err, DefaultColor)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		instruction, err := ParseInstruction(line)
		if err != nil {
			log.Printf("Error parsing line: %s'%s'%s, error: %s%+v%s", SourceColor, line, DefaultColor, ErrorColor, err, DefaultColor)
			continue
		}
		instructions = append(instructions, instruction)
	}
	log.Printf("Read %d instructions from file", len(instructions))
	return instructions, nil
}
func ReadFromArgs() ([]LinkInstruction, error) {
	log.Printf("Reading input from args")
	var instructions []LinkInstruction
	for _, arg := range os.Args[1:] {
		instruction, err := ParseInstruction(arg)
		if err != nil {
			log.Printf("Error parsing arg: %s'%s'%s, error: %s%+v%s", SourceColor, arg, DefaultColor, ErrorColor, err, DefaultColor)
			continue
		}
		instructions = append(instructions, instruction)
	}
	log.Printf("Read %d instructions from args", len(instructions))
	return instructions, nil
}
func ReadFromStdin() ([]LinkInstruction, error) {
	log.Printf("Reading input from stdin")
	var instructions []LinkInstruction

	info, err := os.Stdin.Stat()
	if err != nil {
		log.Fatalf("Failed to stat stdin: %s%+v%s", ErrorColor, err, DefaultColor)
	}
	if info.Mode()&os.ModeNamedPipe != 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			instruction, err := ParseInstruction(line)
			if err != nil {
				log.Printf("Error parsing line: %s'%s'%s, error: %s%+v%s", SourceColor, line, DefaultColor, ErrorColor, err, DefaultColor)
				continue
			}
			instructions = append(instructions, instruction)
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading from stdin: %s%+v%s", ErrorColor, err, DefaultColor)
		}
	}

	log.Printf("Read %d instructions from stdin", len(instructions))
	return instructions, nil
}
