// Package main provides a configurable fake CLI for deterministic subprocess testing.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	quirk := fs.String("quirk", "", "Quirk mode: rate_limit, unauth, gemini_41, hang, spawn_and_hang, noisy")
	exitCode := fs.Int("exit", 0, "Exit code to return")
	stdoutMsg := fs.String("stdout", "", "Message to write to stdout")
	stderrMsg := fs.String("stderr", "", "Message to write to stderr")
	versionFlag := fs.Bool("version", false, "Print version and exit")
	pidFile := fs.String("pid-file", "", "File to write spawned child PID to")
	if err := fs.Parse(os.Args[1:]); err != nil {
		// Unknown flag from a real-CLI invocation pattern (e.g. -p, exec, --json).
		// Simulate a successful headless turn so L2/L3 probe smoke tests pass.
		fmt.Println(`{"ok":true}`)
		os.Exit(0)
	}

	if *versionFlag || hasArg("--version") {
		fmt.Println("fakecli v1.0.0")
		os.Exit(0)
	}

	if hasArg("debug") && hasArg("models") {
		fmt.Println(`{"models":[{"slug":"gpt-6-astra","display_name":"GPT-6-Astra"},{"slug":"gpt-5.5","display_name":"GPT-5.5"}]}`)
		os.Exit(0)
	}

	if *quirk != "" {
		handleQuirk(*quirk, *pidFile)
		return
	}

	if *stdoutMsg != "" {
		fmt.Print(*stdoutMsg)
	}
	if *stderrMsg != "" {
		fmt.Fprint(os.Stderr, *stderrMsg)
	}

	os.Exit(*exitCode)
}

func hasArg(target string) bool {
	for _, a := range os.Args[1:] {
		if a == target {
			return true
		}
	}
	return false
}

func handleQuirk(quirk, pidFile string) {
	switch quirk {
	case "rate_limit", "claude-79500":
		fmt.Println(`{"is_error": true, "error": {"type": "rate_limit_error", "message": "Rate limit reached"}}`)
		os.Exit(0)
	case "unauth", "codex-doctor-exit0":
		fmt.Println(`{"doctor_status": "FAILED", "checks": [{"name": "authentication", "status": "FAILED"}]}`)
		os.Exit(0)
	case "gemini_41", "gemini-base-url-exit41":
		fmt.Fprintln(os.Stderr, "FatalAuthenticationError: invalid credentials")
		os.Exit(41)
	case "hang":
		time.Sleep(24 * time.Hour)
	case "spawn_and_hang":
		handleSpawnAndHang(pidFile)
	case "noisy":
		chunk := strings.Repeat("A", 1024)
		for i := 0; i < 2048; i++ { // 2MB total
			fmt.Println(chunk)
		}
		os.Exit(0)
	case "interactive":
		handleInteractive()
	case "interactive_resize":
		handleInteractiveResize()
	default:
		fmt.Fprintf(os.Stderr, "unknown quirk: %s\n", quirk)
		os.Exit(1)
	}
}

func handleSpawnAndHang(pidFile string) {
	self, err := os.Executable()
	if err != nil {
		self = os.Args[0]
	}
	// #nosec G204
	child := exec.Command(self, "--quirk=hang")
	if err := child.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to spawn child: %v\n", err)
		os.Exit(1)
	}
	if pidFile != "" {
		_ = os.WriteFile(pidFile, []byte(strconv.Itoa(child.Process.Pid)), 0600)
	}
	fmt.Printf("SPAWNED_PID:%d\n", child.Process.Pid)
	time.Sleep(24 * time.Hour)
}

func handleInteractive() {
	fmt.Println("⠋ initializing agent...")
	fmt.Print("Proceed? [y/N]: ")
	var answer string
	_, _ = fmt.Scanln(&answer)
	if strings.ToLower(strings.TrimSpace(answer)) == "y" {
		fmt.Println("CONFIRMED")
		os.Exit(0)
	}
	fmt.Println("ABORTED")
	os.Exit(1)
}

func handleInteractiveResize() {
	fmt.Println("ready for resize")
	sigCh := make(chan os.Signal, 1)
	notifyResize(sigCh)
	go func() {
		<-sigCh
		fmt.Println("RESIZED")
	}()
	var answer string
	_, _ = fmt.Scanln(&answer)
	if strings.ToLower(strings.TrimSpace(answer)) == "y" {
		fmt.Println("CONFIRMED")
	}
	os.Exit(0)
}
