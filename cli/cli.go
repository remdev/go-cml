// Package cli implements the cm command line.
package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/remdev/go-cml/cml"
	"github.com/remdev/go-cml/gen"
	"github.com/remdev/go-cml/refactor"
	"github.com/remdev/go-cml/scl"
)

// Version is the go-cml release. The language matches Context Mapper CML 6.12.
const Version = "0.1.0"

// Main runs the CLI and returns the process exit code.
func Main(args []string) int {
	return Run(os.Stdout, os.Stderr, args)
}

// Run executes the CLI with the provided streams.
func Run(stdout, stderr io.Writer, args []string) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprint(stdout, rootHelp)
		return 0
	}
	if args[0] == "version" || args[0] == "-v" || args[0] == "--version" {
		fmt.Fprintf(stdout, "cm version %s\n", Version)
		return 0
	}
	switch args[0] {
	case "validate":
		return cmdValidate(stdout, stderr, args[1:])
	case "generate":
		return cmdGenerate(stdout, stderr, args[1:])
	case "format":
		return cmdFormat(stdout, stderr, args[1:])
	case "refactor":
		return cmdRefactor(stdout, stderr, args[1:])
	default:
		fmt.Fprintf(stderr, "unknown command %q\n%s", args[0], rootHelp)
		return 2
	}
}

func cmdValidate(stdout, stderr io.Writer, args []string) int {
	opts, rest, err := parseFlags(args, map[string]bool{"i": true, "input": true})
	if err != nil || opts.help {
		fmt.Fprint(stdout, validateHelp)
		if err != nil && !opts.help {
			fmt.Fprintln(stderr, err)
			return 2
		}
		return 0
	}
	if len(rest) > 0 || opts.input == "" {
		fmt.Fprint(stderr, "Missing required option: '--input=<inputPath>'\n")
		fmt.Fprint(stderr, validateHelp)
		return 2
	}
	if !inputExists(opts.input, stderr) {
		return 1
	}
	if strings.EqualFold(filepath.Ext(opts.input), ".scl") {
		src, err := os.ReadFile(opts.input)
		if err != nil {
			fmt.Fprintf(stderr, "ERROR: The file '%s' does not exist.\n", opts.input)
			return 1
		}
		if _, err := scl.Parse(opts.input, string(src)); err != nil {
			fmt.Fprintf(stderr, "ERROR in %s on line 1:%s\n", opts.input, err.Error())
			return 1
		}
		fmt.Fprintf(stdout, "The SCL file '%s' has been validated without errors.\n", opts.input)
		return 0
	}
	if !strings.HasSuffix(strings.ToLower(opts.input), ".cml") {
		fmt.Fprintln(stderr, "ERROR: Please provide a path to a CML (*.cml) file.")
		return 1
	}
	model, err := cml.ParseFile(opts.input)
	if err != nil {
		printSyntax(stderr, opts.input, err)
		return 1
	}
	ds := model.Validate()
	if !cml.HasErrors(ds) {
		fmt.Fprintf(stdout, "The CML file '%s' has been validated without errors.\n", opts.input)
	}
	for _, d := range ds {
		if d.Severity == cml.SevError {
			fmt.Fprintf(stderr, "ERROR in %s on line %d:%s\n", d.Pos.File, d.Pos.Line, d.Message)
			continue
		}
		if d.Severity == cml.SevWarning {
			fmt.Fprintf(stdout, "WARNING in %s on line %d:%s\n", d.Pos.File, d.Pos.Line, d.Message)
		}
	}
	if cml.HasErrors(ds) {
		return 1
	}
	return 0
}

func cmdGenerate(stdout, stderr io.Writer, args []string) int {
	opts, rest, err := parseFlags(args, map[string]bool{
		"i": true, "input": true, "g": true, "generator": true,
		"o": true, "outputDir": true, "t": true, "template": true,
		"f": true, "outputFile": true,
	})
	if err != nil || opts.help {
		fmt.Fprint(stdout, generateHelp)
		if err != nil && !opts.help {
			fmt.Fprintln(stderr, err)
			return 2
		}
		return 0
	}
	if len(rest) > 0 || opts.input == "" {
		fmt.Fprint(stderr, "Missing required option: '--input=<inputPath>'\n")
		fmt.Fprint(stderr, generateHelp)
		return 2
	}
	if opts.generator == "" {
		fmt.Fprintln(stderr, "ERROR: Please provide a generator with -g.")
		fmt.Fprint(stderr, generateHelp)
		return 2
	}
	if opts.output == "" {
		opts.output = "."
	}
	if strings.TrimSpace(opts.output) == "" {
		fmt.Fprintln(stderr, "ERROR: Output directory path is empty.")
		return 1
	}
	if !inputExists(opts.input, stderr) {
		return 1
	}
	if !strings.HasSuffix(strings.ToLower(opts.input), ".cml") {
		fmt.Fprintln(stderr, "ERROR: Please provide a path to a CML (*.cml) file.")
		return 1
	}
	if !outputDirExists(opts.output, stderr) {
		return 1
	}
	if opts.generator == "generic" {
		if opts.template == "" {
			fmt.Fprintln(stderr, "The --template (-t) parameter is required for the 'generic' generator.")
			return 1
		}
		if strings.TrimSpace(opts.outfile) == "" {
			fmt.Fprintln(stderr, "The --outputFile (-f) parameter is required for the 'generic' generator.")
			return 1
		}
	}
	model, err := cml.ParseFile(opts.input)
	if err != nil {
		printSyntax(stderr, opts.input, err)
		return 1
	}
	if ds := model.Validate(); cml.HasErrors(ds) {
		for _, d := range ds {
			if d.Severity == cml.SevError {
				fmt.Fprintf(stderr, "ERROR in %s on line %d:%s\n", d.Pos.File, d.Pos.Line, d.Message)
			}
		}
		return 1
	}
	base := strings.TrimSuffix(filepath.Base(opts.input), filepath.Ext(opts.input))
	var files []gen.File
	switch opts.generator {
	case "plantuml":
		files, err = gen.PlantUML(model, base)
	case "context-map":
		var dot string
		dot, err = gen.ContextMapDOT(model)
		if err == nil {
			files = []gen.File{{Name: base + "_ContextMap.gv", Body: dot}}
		}
	case "mdsl":
		files, err = gen.MDSL(model, base)
	case "scl":
		var body string
		body, err = gen.ServiceCutter(model)
		if err == nil {
			files = []gen.File{{Name: base + ".scl", Body: body}}
		}
	case "generic":
		template, readErr := os.ReadFile(opts.template)
		if readErr != nil {
			fmt.Fprintln(stderr, readErr)
			return 1
		}
		body, renderErr := gen.Render(string(template), model.TemplateData())
		if renderErr != nil {
			fmt.Fprintln(stderr, renderErr)
			return 1
		}
		files = []gen.File{{Name: opts.outfile, Body: body}}
	default:
		fmt.Fprintf(stderr, "unknown generator %q\n%s", opts.generator, generateHelp)
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %s\n", err.Error())
		return 1
	}
	for _, file := range files {
		path := filepath.Join(opts.output, file.Name)
		if err := os.WriteFile(path, []byte(file.Body), 0o644); err != nil {
			fmt.Fprintf(stderr, "ERROR: %s\n", err.Error())
			return 1
		}
	}
	if opts.generator == "context-map" && len(files) == 1 {
		if err := renderGraph(opts.output, files[0]); err != nil {
			fmt.Fprintf(stderr, "ERROR: %s\n", err.Error())
			return 1
		}
	}
	fmt.Fprintf(stdout, "Generated into '%s'.\n", opts.output)
	return 0
}

func renderGraph(dir string, dot gen.File) error {
	dotPath := filepath.Join(dir, dot.Name)
	if _, err := exec.LookPath("dot"); err != nil {
		return fmt.Errorf("graphviz (dot) is required to generate the context map")
	}
	for _, format := range []string{"png", "svg"} {
		out := strings.TrimSuffix(dotPath, ".gv") + "." + format
		cmd := exec.Command("dot", "-T"+format, "-o", out, dotPath)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				return err
			}
			return fmt.Errorf("%w: %s", err, msg)
		}
	}
	return nil
}

func inputExists(path string, stderr io.Writer) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		fmt.Fprintf(stderr, "ERROR: The file '%s' does not exist.\n", path)
		return false
	}
	return true
}

func outputDirExists(path string, stderr io.Writer) bool {
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: Output directory '%s' does not exist.\n", path)
		return false
	}
	if !info.IsDir() {
		fmt.Fprintf(stderr, "ERROR: '%s' is not a directory.\n", path)
		return false
	}
	return true
}

func printSyntax(stderr io.Writer, file string, err error) {
	if syntax, ok := errors.AsType[*cml.SyntaxError](err); ok {
		fmt.Fprintf(stderr, "ERROR in %s on line %d:%s\n", file, syntax.Line, syntax.Msg)
		return
	}
	fmt.Fprintf(stderr, "ERROR in %s on line 1:%s\n", file, err.Error())
}

func cmdFormat(stdout, stderr io.Writer, args []string) int {
	opts, rest, err := parseFlags(args, map[string]bool{"i": true, "input": true, "o": true, "output": true})
	if err != nil || opts.help {
		fmt.Fprint(stdout, formatHelp)
		if err != nil && !opts.help {
			fmt.Fprintln(stderr, err)
			return 2
		}
		return 0
	}
	if len(rest) > 0 || opts.input == "" {
		fmt.Fprint(stderr, formatHelp)
		return 2
	}
	model, err := cml.ParseFile(opts.input)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	text := model.String()
	if opts.output == "" {
		fmt.Fprint(stdout, text)
		return 0
	}
	if err := os.WriteFile(opts.output, []byte(text), 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func cmdRefactor(stdout, stderr io.Writer, args []string) int {
	opts, rest, err := parseFlags(args, map[string]bool{
		"i": true, "input": true, "o": true, "output": true, "r": true, "refactoring": true,
	})
	if err != nil || opts.help || opts.list {
		fmt.Fprint(stdout, refactorHelp())
		if err != nil && !opts.help && !opts.list {
			fmt.Fprintln(stderr, err)
			return 2
		}
		return 0
	}
	if opts.input == "" || opts.refactor == "" {
		fmt.Fprint(stderr, refactorHelp())
		return 2
	}
	model, err := cml.ParseFile(opts.input)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := refactor.Apply(model, opts.refactor, rest); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	text := model.String()
	if opts.output == "" {
		fmt.Fprint(stdout, text)
		return 0
	}
	if err := os.WriteFile(opts.output, []byte(text), 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, opts.output)
	return 0
}

type options struct {
	input, output, generator, template, outfile, refactor string
	help, list                                            bool
}

func parseFlags(args []string, known map[string]bool) (options, []string, error) {
	var opts options
	var rest []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}
		if arg == "-h" || arg == "--help" {
			opts.help = true
			continue
		}
		if arg == "-l" || arg == "--list" {
			opts.list = true
			continue
		}
		name, value, hasValue := cutFlag(arg)
		if name == "" {
			rest = append(rest, arg)
			continue
		}
		if !known[name] && name != "h" && name != "help" && name != "l" && name != "list" {
			return opts, nil, fmt.Errorf("unknown flag %s", arg)
		}
		needsValue := name != "h" && name != "help" && name != "l" && name != "list"
		if needsValue && !hasValue {
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("flag %s requires a value", name)
			}
			i++
			value = args[i]
		}
		switch name {
		case "i", "input":
			opts.input = value
		case "o", "output", "outputDir":
			opts.output = value
		case "g", "generator":
			opts.generator = value
		case "t", "template":
			opts.template = value
		case "f", "outputFile":
			opts.outfile = value
		case "r", "refactoring":
			opts.refactor = value
		}
	}
	return opts, rest, nil
}

func cutFlag(arg string) (name, value string, hasValue bool) {
	if !strings.HasPrefix(arg, "-") || arg == "-" {
		return "", "", false
	}
	body := strings.TrimLeft(arg, "-")
	name, value, hasValue = strings.Cut(body, "=")
	return name, value, hasValue
}

func refactorHelp() string {
	var b strings.Builder
	b.WriteString(refactorUsage)
	for _, cmd := range refactor.Commands() {
		fmt.Fprintf(&b, "  %-44s %s\n    %s\n", cmd.Name+" "+cmd.Use, "", cmd.Help)
	}
	return b.String()
}

const rootHelp = `Context Mapper CLI
usage: cm <command> [options]

  validate    Проверить CML- или SCL-файл
  generate    Сгенерировать диаграммы, контракты или текст
  format      Напечатать канонический CML
  refactor    Применить архитектурный рефакторинг
  version     Показать версию

cm <command> -h выводит справку по команде.
`

const validateHelp = `Usage: cm validate [-hV] -i=<inputPath>
Validates a CML file.

  -h, --help           Показать справку
  -i, --input <file>   CML- или SCL-файл
`

const generateHelp = `Usage: cm generate [-hV] [-f=<outputFileName>] [-g=<generatorType>] [-o=<outputDir>] -i=<inputPath>
Generates output from a CML file.

  -h, --help                 Показать справку
  -i, --input <file>         Входной CML-файл
  -g, --generator <name>     context-map, plantuml, generic, mdsl, scl
  -o, --outputDir <dir>      Каталог для результата, по умолчанию .
  -t, --template <file>      Шаблон для генератора generic
  -f, --outputFile <file>    Имя файла для генератора generic
`

const formatHelp = `Context Mapper CLI
usage: cm format
  -h, --help            Показать справку
  -i, --input <file>    Входной CML-файл
  -o, --output <file>   Куда записать результат; без флага печать в stdout
`

const refactorUsage = `Context Mapper CLI
usage: cm refactor -i <file> -r <name> [args] [-o <file>]
  -h, --help                 Показать справку
  -l, --list                 Список рефакторингов
  -i, --input <file>         Входной CML-файл
  -o, --output <file>        Куда записать модель
  -r, --refactoring <name>   Имя рефакторинга

`
