package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func IsSymlink(path string) (bool, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return false, err
	}

	// os.ModeSymlink is a bitmask that identifies the symlink mode.
	// If the file mode & os.ModeSymlink is non-zero, the file is a symlink.
	return fileInfo.Mode()&os.ModeSymlink != 0, nil
}

func FileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func NormalizePath(input string) string {
	workingdirectory, _ := os.Getwd()
	input = strings.ReplaceAll(input, "\\", "/")
	input = strings.ReplaceAll(input, "\"", "")

	if !filepath.IsAbs(input) {
		input = workingdirectory + "/" + input
	}

	return filepath.Clean(input)
}

func AreSame(lhs string, rhs string) bool {
	lhsinfo, err := os.Stat(lhs)
	if err != nil {
		return false
	}
	rhsinfo, err := os.Stat(rhs)
	if err != nil {
		return false
	}

	return os.SameFile(lhsinfo, rhsinfo)
}

func ConvertHome(input string) (string, error) {
	if strings.Contains(input, "~") {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return input, fmt.Errorf("unable to convert ~ to user directory with error %+v", err)
		}

		return strings.Replace(input, "~", homedir, 1), nil
	}
	return input, nil
}

func GetSyncFilesRecursively(input string, output chan string, status chan error) {
	defer close(output)
	defer close(status)

	var filesProcessed int32
	var foldersProcessed int32
	progressTicker := time.NewTicker(200 * time.Millisecond)
	defer progressTicker.Stop()

	var wg sync.WaitGroup
	var initial sync.Once
	wg.Add(1)
	directories := make(chan string, 100000)
	workerPool := make(chan struct{}, 10000)
	directories <- input

	go func() {
		for {
			fmt.Printf("\rFiles processed: %d; Folders processed: %d; Workers: %d; Directory Stack Size: %d;", filesProcessed, foldersProcessed, len(workerPool), len(directories))
			<-progressTicker.C
		}
	}()

	log.Printf("%+v", len(workerPool))
	go func() {
		for directory := range directories {
			workerPool <- struct{}{}
			wg.Add(1)
			go func(directory string) {
				atomic.AddInt32(&foldersProcessed, 1)
				defer wg.Done()
				defer func() { <-workerPool }()

				files, err := os.ReadDir(directory)
				if err != nil {
					log.Printf("Error reading directory %s: %+v", directory, err)
					return
				}

				for _, file := range files {
					// log.Printf("Processing file %s", file.Name())
					if file.IsDir() {
						directories <- filepath.Join(directory, file.Name())
					} else {
						// log.Println(file.Name(), DirRegex.MatchString(file.Name()))
						if FileRegex.MatchString(file.Name()) {
							log.Printf("Writing")
							output <- filepath.Join(directory, file.Name())
						}
						atomic.AddInt32(&filesProcessed, 1)
					}
				}
				// log.Printf("Done reading directory %s", directory)

				initial.Do(func() {
					// Parallelism is very difficult...
					time.Sleep(250 * time.Millisecond)
					wg.Done()
				})
			}(directory)
		}
	}()

	wg.Wait()
	log.Printf("Done")
}
