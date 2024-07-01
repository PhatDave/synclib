package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func InstructionToString(instruction LinkInstruction) string {
	return fmt.Sprintf("%s%s%s%s%s%s%s%s%s", SourceColor, instruction.Source, DefaultColor, deliminer, TargetColor, instruction.Target, DefaultColor, deliminer, strconv.FormatBool(instruction.Force))
}

func GetSyncFilesRecursively(input string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(input, func(path string, file fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Effectively only find files named "sync" (with no extension!!)
		if !file.IsDir() && DirRegex.MatchString(path) {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
