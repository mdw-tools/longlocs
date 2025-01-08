package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mdw-go/must/jsonmust"
	"github.com/mdw-go/must/osmust"
	"github.com/mdw-go/set/v2/set"
)

var Version = "dev"

func main() {
	var (
		workingDirectory string
		fileExtensions   string
		maxLineLength    int
		verbose          bool
	)
	flags := flag.NewFlagSet(fmt.Sprintf("%s @ %s", filepath.Base(os.Args[0]), Version), flag.ExitOnError)
	flags.StringVar(&workingDirectory, "wd", osmust.Getwd(), "The root directory from which to (recursively) search.")
	flags.StringVar(&fileExtensions, "ext", "", "A comma-separated list of file extensions, without leading dots.")
	flags.IntVar(&maxLineLength, "len", 120, "Any lines longer than this will be reported.")
	flags.BoolVar(&verbose, "v", false, "When set, emit the content of long lines, not just the file:line.")
	_ = flags.Parse(os.Args[1:])

	if fileExtensions == "" {
		log.Fatalln("Must supply one or more file extensions via '-ext' flag.")
	}
	if maxLineLength < 0 {
		log.Fatalln("Must supply a non-negative maximum line length via 'len' flag.")
	}
	extensions := strings.Split(fileExtensions, ",")
	for x := 0; x < len(extensions); x++ {
		extensions[x] = strings.TrimSpace(extensions[x])
	}
	extensionSet := set.FromSeq(slices.Values(extensions))
	log.Println("Searching for long lines in files rooted at:", workingDirectory)
	log.Println("Files considered will end in one of the following extensions:", extensionSet.Slice())
	log.Println("Long lines are those that are longer than:", maxLineLength)

	report := make(map[string]int)
	fileSystem := os.DirFS(workingDirectory)
	fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && filepath.Base(path) == ".git" {
			return fs.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		extension := filepath.Ext(path)
		if strings.HasPrefix(extension, ".") {
			extension = extension[1:]
		}
		if extensionSet.Contains(extension) {
			file, err := fileSystem.Open(path)
			if err != nil {
				return err
			}
			scanner := bufio.NewScanner(file)
			for line := 1; scanner.Scan(); line++ {
				text := scanner.Text()
				length := len(text)
				if length > maxLineLength {
					report[path]++
					content := ""
					if verbose {
						content = "\n" + text
					}
					fmt.Printf("%s:%d (%d chars) %s\n", path, line, length, content)
				}
			}
			return file.Close()
		}
		return nil
	})
	totalCount := 0
	for _, lines := range report {
		totalCount += lines
	}
	finalReport := jsonmust.MarshalIndent(report, "", "  ")
	log.Println("Final Report:", string(finalReport))
	log.Println("Total count of long lines:", totalCount)
}
