package builder

import (
	"fmt"
	"os"
	"os/exec"
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

	if hasBuiltinTools {
		// Run command with internal linker.
		cmd := exec.Command(os.Args[0], append([]string{linker}, flags...)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return execCommand(linker, flags...)
}
