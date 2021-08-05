package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
)

type Line struct {
	Command  string
	Duration time.Duration
}

var maxProcs = runtime.GOMAXPROCS(-1)

var (
	maxLines   = 15
	maxColumns = 200
)

func init() {
	flag.IntVar(&maxLines, "maxlines", maxLines, "maximum lines to print")
	flag.IntVar(&maxColumns, "maxcolumns", maxColumns, "maximum columns to trim")
	flag.Parse()
}

func main() {
	lineIn := make(chan string)
	lineOut := make(chan Line)
	linesOut := make(chan []Line)

	go func() { linesOut <- lineCollector(lineOut) }()

	var wg sync.WaitGroup
	wg.Add(maxProcs)

	for i := 0; i < maxProcs; i++ {
		go func() {
			parseWorker(lineIn, lineOut)
			wg.Done()
		}()
	}

	reader := bufio.NewReaderSize(os.Stdin, bufio.MaxScanTokenSize)
	for {
		line, isPrefix, err := reader.ReadLine()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Fatalln("failed to scan:", err)
			}

			break
		}

		if isPrefix {
			// Line too long; skip.
			continue
		}

		// Do a quick, no alloc check.
		if !bytes.Contains(line, []byte("->")) {
			continue
		}

		lineIn <- string(line)
	}

	close(lineIn)
	wg.Wait()

	close(lineOut)
	lines := <-linesOut

	sort.Slice(lines, func(i, j int) bool {
		// Longest duration first.
		return lines[i].Duration > lines[j].Duration
	})

	switch flag.Arg(0) {
	case "":
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

		fmt.Println("Longest execution sorted by time:")

		for _, line := range lines {
			fmt.Fprintf(w,
				"%s \t| %s\n",
				line.Duration,
				ellipsize(baseCmd(line.Command), maxColumns),
			)
		}

		if err := w.Flush(); err != nil {
			log.Fatalln("failed to flush:", err)
		}

	case "line":
		l, err := strconv.Atoi(flag.Arg(1))
		if err != nil {
			log.Fatalln("failed to parse line number:", err)
		}

		if l < 0 || l >= len(lines) {
			log.Fatalf("line out of bound, must be within 0 < %d < %d", l, len(lines))
		}

		fmt.Println(lines[l].Command)
		fmt.Println(lines[l].Duration)

	default:
		log.Fatalf("unknown subcommand %q", flag.Arg(0))
	}
}

func ellipsize(str string, max int) string {
	if max < 3 {
		max = 80
	}

	if len(str) < max {
		return str
	}

	return str[:max-3] + "..."
}

func baseCmd(command string) string {
	parts := strings.SplitAfterN(command, " ", 2)
	return filepath.Base(parts[0]) + parts[1]
}

func lineCollector(output <-chan Line) []Line {
	var lines []Line
	for out := range output {
		lines = append(lines, out)
	}
	return lines
}

func parseWorker(in <-chan string, out chan<- Line) {
	for str := range in {
		if line, ok := parseLine(str); ok {
			out <- line
		}
	}
}

func parseLine(line string) (Line, bool) {
	parts := strings.Split(line, " -> ")
	if len(parts) < 2 {
		return Line{}, false
	}

	// Join the commands back; spare the last part.
	command := strings.Join(parts[:len(parts)-1], " -> ")

	d, err := time.ParseDuration(parts[len(parts)-1])
	if err != nil {
		log.Printf("invalid duration %q: %v", parts[len(parts)-1], err)
		return Line{}, false
	}

	return Line{
		Command:  command,
		Duration: d,
	}, true
}
