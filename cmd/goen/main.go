package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kamichidu/goen/generator"
	"github.com/kamichidu/goen/internal/asts"
	"github.com/tcnksm/go-latest"
)

var latestSource = &latest.GithubTag{
	Owner:      "kamichidu",
	Repository: "goen",
}

type excludeGlobsFlag []string

func (v *excludeGlobsFlag) Set(s string) error {
	if v == nil {
		*v = []string{}
	}
	*v = append(*v, s)
	return nil
}

func (v excludeGlobsFlag) String() string {
	return strings.Join(v, ", ")
}

func (v excludeGlobsFlag) MakeFilter() func(os.FileInfo) bool {
	return func(info os.FileInfo) bool {
		for _, pat := range v {
			if matched, err := filepath.Match(pat, info.Name()); err != nil {
				log.Panicf("glob match error for pattern (%s): %s", pat, err)
			} else if matched {
				return false
			}
		}
		return true
	}
}

type goCodeFilter interface {
	Filter(filename string, src []byte) ([]byte, error)
}

func isDifferentDir(a string, b string) bool {
	apath, err := filepath.Abs(a)
	if err != nil {
		panic(err)
	}
	bpath, err := filepath.Abs(b)
	if err != nil {
		panic(err)
	}
	return apath != bpath
}

func checkVersion(w io.Writer, verStr string) {
	res, err := latest.Check(latestSource, verStr)
	if err != nil {
		fmt.Fprintf(w, "version check error: %s\n\n", err)
	} else if res.Outdated {
		fmt.Fprintf(w, "%s is not latest, should upgrade to %s\n\n", verStr, res.Current)
	}
}

func run(stdin io.Reader, stdout io.Writer, stderr io.Writer, args []string) int {
	log.SetOutput(stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	checkVersion(stderr, version)

	var (
		outPkgName   string
		outFile      string
		excludeGlobs excludeGlobsFlag
		noGoFmt      bool
		noGoImports  bool
		showVersion  bool
	)
	flg := flag.NewFlagSet(filepath.Base(args[0]), flag.ExitOnError)
	flg.SetOutput(stderr)
	flg.Usage = func() {
		fmt.Fprintf(stderr, "Usage of %s:\n\n", filepath.Base(args[0]))
		fmt.Fprint(stderr, "Positional arguments:\n")
		fmt.Fprint(stderr, "  directory\n")
		fmt.Fprint(stderr, "      parsing directory (default \".\")\n\n")
		fmt.Fprint(stderr, "Optional arguments:\n")
		flg.PrintDefaults()
	}
	flg.BoolVar(&showVersion, "v", false, "show version")
	flg.StringVar(&outPkgName, "p", "", "output package name (default: auto detect)")
	flg.StringVar(&outFile, "o", "-", "output filename")
	flg.Var(&excludeGlobs, "x", "exclude file globs")
	flg.BoolVar(&noGoFmt, "no-gofmt", false, "generating go codes without gofmt")
	flg.BoolVar(&noGoImports, "no-goimports", false, "generating go codes without goimports")
	if err := flg.Parse(args[1:]); err != nil {
		log.Print(err)
		flg.Usage()
		return 128
	} else if showVersion {
		fmt.Fprintf(stdout, "%s - %s\n", filepath.Base(args[0]), version)
		return 0
	} else if flg.NArg() > 1 {
		flg.Usage()
		return 128
	}
	srcDir := flg.Arg(0)
	if srcDir == "" {
		srcDir = "."
	}

	var out io.Writer
	if outFile == "-" {
		out = stdout
	} else {
		outDir := filepath.Dir(outFile)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			log.Printf("failed to create directory (%s): %s", outDir, err)
			return 1
		}
		wc, err := os.Create(outFile)
		if err != nil {
			log.Printf("failed to create file (%s): %s", outFile, err)
			return 1
		}
		defer wc.Close()

		// avoiding invalid go file while generating this file
		if _, err := io.WriteString(wc, "// +build ignore\n\npackage _"); err != nil {
			log.Printf("failed to write go build tag: %s", err)
			return 1
		}

		if _, err := wc.Seek(0, io.SeekStart); err != nil {
			log.Printf("failed to rewind: %s", err)
			return 1
		}

		out = wc
		if err := excludeGlobs.Set(filepath.Base(outFile)); err != nil {
			log.Panic(err)
		}
	}

	// if output to another package
	if outFile != "-" {
		outDir := filepath.Dir(outFile)
		if isDifferentDir(srcDir, outDir) {
			pkgName, _ := asts.AssumeImport(outDir)
			if outPkgName != "" && pkgName != outPkgName {
				log.Printf("output package name conflict: given %q but is %q", outPkgName, pkgName)
				return 128
			}
			outPkgName = pkgName
		}
	}

	gen := &generator.Generator{
		SrcDir:        srcDir,
		SrcFileFilter: excludeGlobs.MakeFilter(),
		OutPkgName:    outPkgName,
	}
	if err := gen.ParseDir(); err != nil {
		log.Printf("parse go files error: %s", err)
		return 1
	}
	buf := new(bytes.Buffer)
	if err := gen.Generate(buf); err != nil {
		log.Printf("generating go files error: %s", err)
		return 1
	}

	filters := []goCodeFilter{}
	if !noGoFmt {
		filters = append(filters, new(goFmtFilter))
	}
	if !noGoImports {
		filters = append(filters, new(goImportsFilter))
	}
	src := buf.Bytes()
	for _, filter := range filters {
		res, err := filter.Filter(outFile, src)
		if err != nil {
			log.Printf("applying go code filter error: %s", err)
			res = src
		}
		src = res
	}

	if _, err := out.Write(src); err != nil {
		log.Printf("failed to write output: %s", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(run(os.Stdin, os.Stdout, os.Stderr, os.Args))
}
