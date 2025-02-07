package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	extList  = flag.String("ext", "", "Comma-separated file extensions to include (empty = all files)")
	allFiles = flag.Bool("all", false, "Format all files recursively")
	current  = flag.Bool("current", false, "Format only changed files in the current branch")
)

func main() {
	flag.Parse()

	if flag.Arg(0) == "help" {
		printHelp()
		return
	}

	if *allFiles {
		if !confirmAction("This will format all files recursively in the current directory. Do you want to continue? (yes/no): ") {
			fmt.Println("Operation canceled.")
			return
		}
		formatAllFiles()
		return
	}

	gitRoot, err := getGitRoot()
	if err != nil {
		log.Fatalf("Error finding Git repository: %v", err)
	}

	var files []string
	if *current {
		files, err = getChangedFiles(gitRoot)
		if err != nil {
			log.Fatalf("Error getting changed files: %v", err)
		}
	} else {
		fmt.Println("No valid option selected. Use --current or --all.")
		return
	}

	extensions := strings.Split(*extList, ",")
	if *extList == "" {
		extensions = []string{".js", ".ts", ".json", ".tsx", ".jsx"}
	}

	filtered := filterFiles(files, extensions)
	if len(filtered) == 0 {
		fmt.Println("No files to format")
		return
	}

	if err := runPrettier(filtered); err != nil {
		log.Fatalf("Error formatting files: %v", err)
	}

	fmt.Printf("Successfully formatted %d files:\n", len(filtered))
	for _, file := range filtered {
		fmt.Println(" ", file)
	}
}

func getGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func getChangedFiles(gitRoot string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	var files []string
	for _, file := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if file == "" {
			continue
		}
		files = append(files, filepath.Join(gitRoot, file))
	}
	return files, nil
}

func filterFiles(files, exts []string) []string {
	var filtered []string
	includeAll := false

	for _, ext := range exts {
		if ext == "" {
			includeAll = true
			break
		}
	}

	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue
		}

		if includeAll {
			filtered = append(filtered, file)
			continue
		}

		for _, ext := range exts {
			if strings.HasSuffix(file, ext) {
				filtered = append(filtered, file)
				break
			}
		}
	}
	return filtered
}

func runPrettier(files []string) error {
	args := append([]string{"--write"}, files...)
	cmd := exec.Command("prettier", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("prettier exited with code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("failed to run prettier: %w (make sure it's installed and in PATH)", err)
	}
	return nil
}

func formatAllFiles() {
	cmd := exec.Command("prettier", "--write", "./**/*")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("Error formatting all files: %v", err)
	}
	fmt.Println("Successfully formatted all files.")
}

func confirmAction(prompt string) bool {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return response == "yes"
}

func printHelp() {
	fmt.Println("Usage: pretti [options]")
	fmt.Println("Options:")
	fmt.Println("  --ext <exts>     Comma-separated file extensions to include (default: .js, .ts, .json, .tsx, .jsx)")
	fmt.Println("  --all            Format all files recursively in the current directory (asks for confirmation)")
	fmt.Println("  --current        Format only changed files in the current branch")
	fmt.Println("  help             Show this help message")
}
