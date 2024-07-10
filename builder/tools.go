package builder

import (
	"bytes"
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// runCCompiler invokes a C compiler with the given arguments.
func runCCompiler(flags ...string) error {
	if hasBuiltinTools {
		// Compile this with the internal Clang compiler.
		cmd := exec.Command(os.Args[0], append([]string{"clang"}, flags...)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Compile this with an external invocation of the Clang compiler.
	return execCommand("clang", flags...)
}

// link invokes a linker with the given name and flags.
func link(linker string, flags ...string) error {
	// We only support LLD.
	if linker != "ld.lld" && linker != "wasm-ld" {
		return fmt.Errorf("unexpected: linker %s should be ld.lld or wasm-ld", linker)
	}

	var cmd *exec.Cmd
	if hasBuiltinTools {
		cmd = exec.Command(os.Args[0], append([]string{linker}, flags...)...)
	} else {
		name, err := LookupCommand(linker)
		if err != nil {
			return err
		}
		cmd = exec.Command(name, flags...)
	}
	var buf bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &buf
	err := cmd.Run()
	if err != nil {
		if buf.Len() == 0 {
			return fmt.Errorf("failed to run linker: %w", err)
		}
		return parseLLDErrors(buf.String())
	}
	return nil
}

// Split LLD errors into individual erros (including errors that continue on the
// next line, using a ">>>" prefix). If possible, replace the raw errors with a
// more user-friendly version (and one that's more in a Go style).
func parseLLDErrors(text string) error {
	// Split linker output in separate error messages.
	lines := strings.Split(text, "\n")
	var errorLines []string // one or more line (belonging to a single error) per line
	for _, line := range lines {
		if len(errorLines) != 0 && strings.HasPrefix(line, ">>> ") {
			errorLines[len(errorLines)-1] = errorLines[len(errorLines)-1] + "\n" + line[4:]
			continue
		}
		errorLines = append(errorLines, line)
	}

	// Parse error messages.
	var linkErrors []error
	for _, message := range errorLines {
		parsedError := false

		// Check for undefined symbols.
		// This can happen in some cases like with CGo and //go:linkname tricker.
		if matches := regexp.MustCompile(`^ld.lld: error: undefined symbol: (.*)\n`).FindStringSubmatch(message); matches != nil {
			symbolName := matches[1]
			for _, line := range strings.Split(message, "\n") {
				matches := regexp.MustCompile(`referenced by .* \(((.*):([0-9]+))\)`).FindStringSubmatch(line)
				if matches != nil {
					parsedError = true
					line, _ := strconv.Atoi(matches[3])
					// TODO: detect common mistakes like -gc=none?
					linkErrors = append(linkErrors, scanner.Error{
						Pos: token.Position{
							Filename: matches[2],
							Line:     line,
						},
						Msg: "linker could not find symbol " + symbolName,
					})
				}
			}
		}

		// If we couldn't parse the linker error: show the error as-is to
		// the user.
		if !parsedError {
			linkErrors = append(linkErrors, LinkerError{message})
		}
	}
	return newMultiError(linkErrors)
}

// LLD linker error that could not be parsed.
type LinkerError struct {
	Msg string
}

func (e LinkerError) Error() string {
	return e.Msg
}

// Return the error as returned by LLD (without the ">>> " removed).
func (e LinkerError) OriginalError() string {
	return strings.ReplaceAll(e.Msg, "\n", "\n>>> ")
}
