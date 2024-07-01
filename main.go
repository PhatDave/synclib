package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"regexp"
	"sync"
	"sync/atomic"
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
var FileRegex, _ = regexp.Compile(`^sync$`)
var programName = os.Args[0]

func main() {
	recurse := flag.String("r", "", "recurse into directories")
	file := flag.String("f", "", "file to read instructions from")
	debug := flag.Bool("d", false, "debug")
	flag.Parse()

	if *debug {
		log.SetFlags(log.Lmicroseconds | log.Lshortfile)
		logFile, err := os.Create("main.log")
		if err != nil {
			log.Printf("Error creating log file: %v", err)
			os.Exit(1)
		}
		logger := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(logger)
	} else {
		log.SetFlags(log.Lmicroseconds)
	}

	log.Printf("Recurse: %s", *recurse)
	log.Printf("File: %s", *file)

	instructions := make(chan LinkInstruction, 1000)
	status := make(chan error)
	if *recurse != "" {
		go ReadFromFilesRecursively(*recurse, instructions, status)
	} else if *file != "" {
		go ReadFromFile(*file, instructions, status, true)
	} else if len(os.Args) > 1 {
		go ReadFromArgs(instructions, status)
	} else {
		go ReadFromStdin(instructions, status)
	}

	go func() {
		for {
			err, ok := <-status
			if !ok {
				break
			}
			if err != nil {
				log.Println(err)
			}
		}
	}()

	var instructionsDone int32
	var wg sync.WaitGroup
	for {
		instruction, ok := <-instructions
		if !ok {
			log.Printf("No more instructions to process")
			break
		}
		log.Printf("Processing: %s", instruction.String())
		status := make(chan error)
		go instruction.RunAsync(status)
		wg.Add(1)
		err := <-status
		if err != nil {
			log.Printf("Failed parsing instruction %s%s%s due to %s%+v%s", SourceColor, instruction.String(), DefaultColor, ErrorColor, err, DefaultColor)
		}
		atomic.AddInt32(&instructionsDone, 1)
		wg.Done()
	}
	wg.Wait()
	log.Println("All done")
	if instructionsDone == 0 {
		log.Printf("No input provided")
		log.Printf("Provide input as: ")
		log.Printf("Arguments - %s <source>,<target>,<force?>", programName)
		log.Printf("File - %s -f <file>", programName)
		log.Printf("Folder (finding sync files in folder recursively) - %s -r <folder>", programName)
		log.Printf("stdin - (cat <file> | %s)", programName)
		os.Exit(1)
	}
}

func ReadFromFilesRecursively(input string, output chan LinkInstruction, status chan error) {
	defer close(output)
	defer close(status)

	input = NormalizePath(input)
	log.Printf("Reading input from files recursively starting in %s%s%s", PathColor, input, DefaultColor)

	files := make(chan string, 128)
	recurseStatus := make(chan error)
	go GetSyncFilesRecursively(input, files, recurseStatus)
	go func() {
		for {
			err, ok := <-recurseStatus
			if !ok {
				break
			}
			if err != nil {
				log.Printf("Failed to get sync files recursively: %s%+v%s", ErrorColor, err, DefaultColor)
				status <- err
			}
		}
	}()

	var wg sync.WaitGroup
	for {
		file, ok := <-files
		if !ok {
			log.Printf("No more files to process")
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println(file)
			file = NormalizePath(file)
			log.Printf("Processing file: %s%s%s", PathColor, file, DefaultColor)

			// This "has" to be done because instructions are resolved in relation to cwd
			fileDir := DirRegex.FindStringSubmatch(file)
			if fileDir == nil {
				log.Printf("Failed to extract directory from %s%s%s", SourceColor, file, DefaultColor)
				return
			}
			log.Printf("Changing directory to %s%s%s (for %s%s%s)", PathColor, fileDir[1], DefaultColor, PathColor, file, DefaultColor)
			err := os.Chdir(fileDir[1])
			if err != nil {
				log.Printf("Failed to change directory to %s%s%s: %s%+v%s", SourceColor, fileDir[1], DefaultColor, ErrorColor, err, DefaultColor)
				return
			}

			ReadFromFile(file, output, status, false)
		}()
	}
	wg.Wait()
}
func ReadFromFile(input string, output chan LinkInstruction, status chan error, doclose bool) {
	if doclose {
		defer close(output)
		defer close(status)
	}

	input = NormalizePath(input)
	log.Printf("Reading input from file: %s%s%s", PathColor, input, DefaultColor)
	file, err := os.Open(input)
	if err != nil {
		log.Fatalf("Failed to open file %s%s%s: %s%+v%s", SourceColor, input, DefaultColor, ErrorColor, err, DefaultColor)
		return
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
		log.Printf("Read instruction: %s", instruction.String())
		output <- instruction
	}
}
func ReadFromArgs(output chan LinkInstruction, status chan error) {
	defer close(output)
	defer close(status)

	log.Printf("Reading input from args")
	for _, arg := range os.Args[1:] {
		instruction, err := ParseInstruction(arg)
		if err != nil {
			log.Printf("Error parsing arg: %s'%s'%s, error: %s%+v%s", SourceColor, arg, DefaultColor, ErrorColor, err, DefaultColor)
			continue
		}
		output <- instruction
	}
}
func ReadFromStdin(output chan LinkInstruction, status chan error) {
	defer close(output)
	defer close(status)

	log.Printf("Reading input from stdin")

	info, err := os.Stdin.Stat()
	if err != nil {
		log.Fatalf("Failed to stat stdin: %s%+v%s", ErrorColor, err, DefaultColor)
		status <- err
		return
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
			output <- instruction
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading from stdin: %s%+v%s", ErrorColor, err, DefaultColor)
			status <- err
			return
		}
	}
}
