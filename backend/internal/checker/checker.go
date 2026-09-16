// Package checker executes kubectl commands against the underlying cluster
// (minikube) to prepare task environments, grade answers with weighted
// checks (killer.sh style), and clean up afterwards.
package checker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
)

const (
	DefaultBinary  = "kubectl"
	DefaultTimeout = 20 * time.Second
	// maxOutput keeps stored kubectl output small.
	maxOutput = 512
	// execTimeout bounds a single terminal command.
	execTimeout = 30 * time.Second
	// maxExecOutput caps terminal output sent back to the browser.
	maxExecOutput = 16 * 1024
)

// Checker runs kubectl commands using the current kubeconfig context.
type Checker struct {
	Binary  string
	Timeout time.Duration
	// homeDir is the sandbox used as the candidate's working directory.
	// Files created with the built-in vi/nano editor live here, and every
	// terminal command executes with this directory as its cwd.
	homeDir string
}

// New returns a Checker with defaults (override via fields if needed).
func New() *Checker {
	home := filepath.Join(os.TempDir(), "ckad-simulator", "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		home, _ = os.Getwd()
	}
	return &Checker{Binary: DefaultBinary, Timeout: DefaultTimeout, homeDir: home}
}

// HomeDir returns the sandbox directory serving as the candidate's home.
func (c *Checker) HomeDir() string { return c.homeDir }

// ResolvePath maps a candidate-visible path onto the sandbox filesystem.
// Absolute paths are interpreted relative to the exam home (chroot-style),
// and any path escaping the sandbox is rejected.
func (c *Checker) ResolvePath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		p = "."
	}
	rel := strings.TrimPrefix(filepath.Clean(p), "/")
	full := filepath.Join(c.homeDir, rel)
	if full != c.homeDir && !strings.HasPrefix(full, c.homeDir+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes the exam home: %s", p)
	}
	return full, nil
}

// ReadFile loads a file from the exam sandbox (used by the built-in editors).
func (c *Checker) ReadFile(p string) (string, error) {
	full, err := c.ResolvePath(p)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", p)
		}
		return "", err
	}
	return string(b), nil
}

// WriteFile stores a file inside the exam sandbox (built-in editors).
func (c *Checker) WriteFile(p, content string) error {
	full, err := c.ResolvePath(p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), 0o644)
}

// CompleteLine provides shell-style Tab completion for an exam terminal
// line. If the line is a lone word it completes command names from the
// allow-list; otherwise it completes the last word against sandbox paths,
// appending "/" for directories.
func (c *Checker) CompleteLine(line string) []string {
	raw := line
	line = strings.TrimSpace(line)
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}
	// A trailing space means the candidate started a new (empty) argument.
	completingNewArg := strings.HasSuffix(raw, " ")
	if len(fields) == 1 && !completingNewArg {
		// Completing the command itself. All known binaries are offered,
		// including vi/nano which the browser terminal intercepts.
		prefix := line
		var cmds []string
		for bin := range allowedPrefixes {
			if strings.HasPrefix(bin, prefix) {
				cmds = append(cmds, bin)
			}
		}
		sort.Strings(cmds)
		return cmds
	}

	// Completing a path argument: take the trailing word verbatim (may be
	// partially typed, may end with a slash).
	last := ""
	if !completingNewArg {
		last = fields[len(fields)-1]
	}
	dirPart := ""
	base := last
	if i := strings.LastIndex(last, "/"); i >= 0 {
		dirPart = last[:i+1]
		base = last[i+1:]
	}
	full, err := c.ResolvePath(strings.TrimSuffix(dirPart, "/"))
	if err != nil {
		return nil
	}
	if dirPart == "" {
		full = c.homeDir
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return nil
	}
	var matches []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, base) {
			continue
		}
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(base, ".") {
			continue // skip hidden files unless explicitly requested
		}
		m := dirPart + name
		if e.IsDir() {
			m += "/"
		}
		matches = append(matches, m)
	}
	sort.Strings(matches)
	return matches
}

// Grade runs every check of a question against the live cluster and
// returns per-check results with partial credit.
func (c *Checker) Grade(ctx context.Context, q *models.Question) []models.CheckResult {
	results := make([]models.CheckResult, 0, len(q.Checks))
	for i := range q.Checks {
		results = append(results, c.runCheck(ctx, &q.Checks[i]))
	}
	return results
}

func (c *Checker) runCheck(ctx context.Context, chk *models.Check) models.CheckResult {
	res := models.CheckResult{
		CheckID:     chk.ID,
		Description: chk.Description,
		MaxPoints:   chk.Weight,
	}

	cctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	out, err := exec.CommandContext(cctx, c.Binary, strings.Fields(chk.CommandArgs)...).CombinedOutput()
	res.Output = truncate(strings.TrimSpace(string(out)))

	met := err == nil && expectationMet(chk, res.Output)
	if chk.Invert {
		met = !met
	}
	res.Passed = met
	if met {
		res.Points = chk.Weight
	}
	return res
}

// expectationMet evaluates stdout against the check's expectations.
func expectationMet(chk *models.Check, output string) bool {
	switch {
	case chk.ExpectSubstring != "":
		return strings.Contains(strings.ToLower(output), strings.ToLower(chk.ExpectSubstring))
	case chk.ExpectRegex != "":
		re, err := regexp.Compile(chk.ExpectRegex)
		if err != nil {
			return false
		}
		return re.MatchString(output)
	default:
		// No explicit expectation: pass iff the command succeeded.
		return true
	}
}

// Prepare provisions the environment for a set of questions (best effort).
// It returns a human-readable log of each step's outcome.
func (c *Checker) Prepare(ctx context.Context, qs []*models.Question) []string {
	var logs []string
	for _, q := range qs {
		for _, step := range q.Prepare {
			logs = append(logs, c.runStep(ctx, q.ID, step)...)
		}
	}
	return logs
}

func (c *Checker) runStep(ctx context.Context, questionID string, step models.SetupStep) []string {
	// A file step touches the sandbox, not the cluster.
	if step.File != "" {
		status := "ok"
		detail := step.File
		if err := c.WriteFile(step.File, step.FileContent); err != nil {
			status = "failed"
			detail = err.Error()
		}
		return []string{fmt.Sprintf("[%s] %s: %s — %s", questionID, step.Name, status, detail)}
	}

	var args []string
	switch {
	case step.YAML != "":
		args = []string{"apply", "-f", "-"}
		if step.Namespace != "" {
			args = append(args, "-n", step.Namespace)
		}
	case step.CommandArgs != "":
		args = strings.Fields(step.CommandArgs)
	default:
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, c.Binary, args...)
	if step.YAML != "" {
		cmd.Stdin = strings.NewReader(step.YAML)
	}
	out, err := cmd.CombinedOutput()
	status := "ok"
	if err != nil {
		status = "failed"
	}
	return []string{fmt.Sprintf("[%s] %s: %s — %s", questionID, step.Name, status, truncate(strings.TrimSpace(string(out))))}
}

// Cleanup resets cluster state by running each command verbatim as
// `kubectl <args>` (best effort). Commands should be idempotent deletes,
// e.g. "delete namespace ckad-pods --ignore-not-found".
func (c *Checker) Cleanup(ctx context.Context, cmds []string) []string {
	var logs []string
	for _, cm := range cmds {
		cctx, cancel := context.WithTimeout(ctx, c.Timeout)
		out, err := exec.CommandContext(cctx, c.Binary, strings.Fields(cm)...).CombinedOutput()
		cancel()
		status := "ok"
		if err != nil {
			status = "skipped"
		}
		logs = append(logs, fmt.Sprintf("%s — %s", status, truncate(strings.TrimSpace(string(out)))))
	}
	return logs
}

// protectedNamespaces are never deleted by a reset, even if a question
// happens to name one. Losing kube-system to an exam reset would take the
// cluster with it.
var protectedNamespaces = map[string]bool{
	"default": true, "kube-system": true, "kube-public": true,
	"kube-node-lease": true,
}

// ResetCluster deletes every exam namespace still lingering in the cluster
// in a single kubectl call. `owned` lists the namespaces the question bank
// provisions — only those (plus anything prefixed "ckad-") are collected,
// so namespaces belonging to whoever else uses the cluster are left alone.
//
// Passing the owned set in matters: exam questions name their namespaces
// freely ("q34", "ap-11", "ingress-namespace"), so a prefix rule alone
// collects nothing and stale resources survive into the next exam, where
// they score points nobody earned.
func (c *Checker) ResetCluster(ctx context.Context, owned []string) {
	cctx, cancel := context.WithTimeout(ctx, c.Timeout*3)
	defer cancel()

	out, err := exec.CommandContext(cctx, c.Binary, "get", "namespace", "-o", "name").CombinedOutput()
	if err != nil {
		return
	}

	ownedSet := make(map[string]bool, len(owned))
	for _, n := range owned {
		ownedSet[strings.TrimSpace(n)] = true
	}

	var toDelete []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Lines look like "namespace/q34".
		name := strings.TrimPrefix(line, "namespace/")
		if protectedNamespaces[name] || strings.HasPrefix(name, "kube-") {
			continue
		}
		if ownedSet[name] || strings.HasPrefix(name, "ckad-") {
			toDelete = append(toDelete, name)
		}
	}

	if len(toDelete) == 0 {
		return
	}

	// Single bulk delete — much faster than one-by-one.
	args := append([]string{"delete", "namespace"}, toDelete...)
	args = append(args, "--ignore-not-found", "--wait=false")
	bulkCtx, bulkCancel := context.WithTimeout(ctx, c.Timeout*3)
	defer bulkCancel()
	_ = exec.CommandContext(bulkCtx, c.Binary, args...).Run()
}

const (
	// namespaceDeleteTimeout bounds how long a new exam waits for the
	// previous one's namespaces to finish terminating.
	namespaceDeleteTimeout = 90 * time.Second
	// namespacePollInterval is how often that wait re-checks.
	namespacePollInterval = time.Second
)

// WaitNamespacesGone blocks until the given namespaces have finished
// deleting. ResetCluster deletes without waiting (it must stay fast), so a
// new exam would otherwise race the previous one's teardown: creating a
// resource in a Terminating namespace is rejected, leaving the task
// unprepared. Bounded: on timeout it returns and lets Prepare report
// whatever it finds.
//
// This polls `kubectl get` rather than using `kubectl wait --for=delete`,
// which errors on already-absent resources and whose --ignore-not-found
// flag does not exist across kubectl versions.
func (c *Checker) WaitNamespacesGone(ctx context.Context, names []string) {
	targets := make([]string, 0, len(names))
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" && !protectedNamespaces[n] {
			targets = append(targets, n)
		}
	}
	if len(targets) == 0 {
		return
	}

	args := append([]string{"get", "namespace", "--ignore-not-found", "-o", "name"}, targets...)
	deadline := time.Now().Add(namespaceDeleteTimeout)
	for {
		cctx, cancel := context.WithTimeout(ctx, c.Timeout)
		out, err := exec.CommandContext(cctx, c.Binary, args...).Output()
		cancel()
		if err == nil && len(bytes.TrimSpace(out)) == 0 {
			return // all gone
		}
		if time.Now().After(deadline) {
			return // give up rather than block the exam indefinitely
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(namespacePollInterval):
		}
	}
}

// ClusterStatus verifies connectivity to the cluster and returns the
// control-plane line from `kubectl cluster-info`.
func (c *Checker) ClusterStatus(ctx context.Context) (connected bool, detail string) {
	cctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	out, err := exec.CommandContext(cctx, c.Binary, "cluster-info").CombinedOutput()
	detail = truncate(strings.TrimSpace(string(out)))
	if err != nil {
		return false, detail
	}
	lines := strings.Split(detail, "\n")
	if len(lines) > 0 && lines[0] != "" {
		detail = lines[0]
	}
	return true, detail
}

// allowedPrefixes are the binaries a candidate may run in the exam terminal.
// The real CKAD environment exposes kubectl plus basic shell tools; here we
// keep the surface small and safe while feeling like the real thing.
var allowedPrefixes = map[string]bool{
	"kubectl": true,
	"k":       true, // common kubectl alias

	// Inspection & text processing.
	"cat": true, "ls": true, "grep": true, "echo": true, "printf": true,
	"jq": true, "yq": true, "curl": true, "git": true,
	"head": true, "tail": true, "wc": true, "sort": true, "uniq": true,
	"diff": true, "cut": true, "tr": true, "tac": true, "rev": true,
	"sed": true, "awk": true, "find": true, "base64": true,
	"stat": true, "du": true, "df": true,

	// Filesystem manipulation inside the exam home.
	"pwd": true, "mkdir": true, "touch": true, "rm": true, "cp": true, "mv": true,

	// Misc utilities.
	"env": true, "which": true, "date": true, "hostname": true,
	"whoami": true, "sleep": true, "seq": true, "basename": true, "dirname": true,

	// Interactive editors are emulated by the browser terminal; they never
	// reach the backend in normal use. Kept here to return a helpful hint if
	// the API is called directly.
	"vi": false, "vim": false, "view": false, "nano": false,
	"clear": false, // handled client-side
}

// pathFreeCommands never open a file named by their arguments, so those
// arguments are passed through exactly as typed — re-rooting them would
// corrupt `echo a/b` or `basename /usr/local/bin`.
var pathFreeCommands = map[string]bool{
	"echo": true, "printf": true, "seq": true, "date": true, "sleep": true,
	"whoami": true, "hostname": true, "env": true, "which": true,
	"tr": true, "rev": true, "basename": true, "dirname": true,
}

// patternCommands take a search pattern or a script as their first non-flag
// argument. It must not be treated as a path, or `kubectl get ns -o name |
// grep namespace/q34` and `sed 's/a/b/'` would break.
var patternCommands = map[string]bool{"grep": true, "sed": true, "awk": true}

// kubectlFileFlags are the kubectl flags whose value names a file on the
// exam host. Every other kubectl argument is left untouched: object names,
// JSONPath expressions and the paths in `kubectl exec pod -- cat /etc/x`
// refer to the cluster or to a pod's filesystem, not to the host.
var kubectlFileFlags = map[string]bool{
	"-f": true, "--filename": true,
	"-k": true, "--kustomize": true,
	"--from-file": true, "--from-env-file": true,
	"--kubeconfig":            true,
	"--client-certificate":    true,
	"--client-key":            true,
	"--certificate-authority": true,
	"--patch-file":            true,
	"--cert":                  true,
	"--key":                   true,
}

// isFlag reports whether a token is an option rather than an operand.
// A lone "-" is the conventional stdin placeholder, not a flag.
func isFlag(a string) bool { return len(a) > 1 && strings.HasPrefix(a, "-") }

// refersToPath reports whether an argument names a location on the
// filesystem. Bare names ("pod.yaml") need no rewriting: commands run with
// the exam home as their working directory, so they already resolve inside
// the sandbox. Only arguments that navigate the tree can leave it.
func refersToPath(a string) bool {
	return strings.Contains(a, "/") || a == "." || a == ".."
}

// confineArgs rewrites every argument naming a file on the exam host so it
// resolves inside the sandbox, and rejects any argument that would escape.
// Absolute paths are re-rooted chroot-style, exactly as the built-in
// editors treat them (see ResolvePath), so `/pod.yaml` in the terminal and
// `/pod.yaml` in vi are the same file.
func (c *Checker) confineArgs(bin string, args []string) ([]string, error) {
	if pathFreeCommands[bin] {
		return args, nil
	}
	if bin == c.Binary || bin == DefaultBinary {
		return c.confineKubectlArgs(args)
	}

	out := make([]string, len(args))
	copy(out, args)
	seenOperand := false
	for i, a := range out {
		if isFlag(a) {
			// `--flag=/path` still carries a path in its value.
			if name, value, found := strings.Cut(a, "="); found && refersToPath(value) {
				full, err := c.ResolvePath(value)
				if err != nil {
					return nil, err
				}
				out[i] = name + "=" + full
			}
			continue
		}

		firstOperand := !seenOperand
		seenOperand = true

		if strings.Contains(a, "://") {
			if strings.HasPrefix(strings.ToLower(a), "file://") {
				return nil, fmt.Errorf("local file URLs are not available in the exam terminal: %s", a)
			}
			continue // a network URL, not a path on this host
		}
		// The pattern/script operand is not a path — unless it is spelled
		// like one ("../x"), in which case it must still be confined.
		if firstOperand && patternCommands[bin] && !strings.HasPrefix(a, "/") &&
			!strings.HasPrefix(a, "./") && !strings.HasPrefix(a, "../") {
			continue
		}
		if !refersToPath(a) {
			continue
		}
		full, err := c.ResolvePath(a)
		if err != nil {
			return nil, err
		}
		out[i] = full
	}
	return out, nil
}

// confineKubectlArgs confines only the kubectl arguments that name a file on
// the exam host: the values of kubectlFileFlags, and the local side of
// `kubectl cp`.
func (c *Checker) confineKubectlArgs(args []string) ([]string, error) {
	out := make([]string, len(args))
	copy(out, args)

	isCopy := false
	for _, a := range out {
		if !isFlag(a) {
			isCopy = a == "cp"
			break
		}
	}

	for i := 0; i < len(out); i++ {
		a := out[i]
		if name, value, found := strings.Cut(a, "="); found && kubectlFileFlags[name] {
			conf, err := c.confineFlagValue(value)
			if err != nil {
				return nil, err
			}
			out[i] = name + "=" + conf
			continue
		}
		if kubectlFileFlags[a] && i+1 < len(out) {
			conf, err := c.confineFlagValue(out[i+1])
			if err != nil {
				return nil, err
			}
			out[i+1] = conf
			i++
			continue
		}
		// `kubectl cp <local> pod:/remote` — the operand without a colon is
		// the one that touches this host.
		if isCopy && !isFlag(a) && !strings.Contains(a, ":") && refersToPath(a) {
			full, err := c.ResolvePath(a)
			if err != nil {
				return nil, err
			}
			out[i] = full
		}
	}
	return out, nil
}

// confineFlagValue confines the path carried by a kubectl flag value,
// preserving the `--from-file=key=/path` spelling and the "-" stdin
// placeholder.
func (c *Checker) confineFlagValue(v string) (string, error) {
	if v == "" || v == "-" {
		return v, nil
	}
	if key, path, found := strings.Cut(v, "="); found {
		full, err := c.ResolvePath(path)
		if err != nil {
			return "", err
		}
		return key + "=" + full, nil
	}
	full, err := c.ResolvePath(v)
	if err != nil {
		return "", err
	}
	return full, nil
}

// ExecResult is the outcome of one terminal command execution.
type ExecResult struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exitCode"`
}

// Exec runs a command line from the exam terminal against the cluster.
// Supported shell features:
//   - pipes chaining multiple allow-listed commands (e.g.
//     `kubectl get pods | grep web`)
//   - output redirection to files inside the exam home (`> file`, `>> file`)
//   - every command runs with the exam home as its working directory
//
// Interactive editors (vi/vim/nano) are emulated by the browser terminal and
// never reach this function in normal use.
func (c *Checker) Exec(ctx context.Context, command string) ExecResult {
	command = strings.TrimSpace(command)
	if command == "" {
		return ExecResult{}
	}

	parts := splitTopLevel(command, '|')
	if len(parts) == 0 {
		return ExecResult{Output: shellLikeUnsupported(command), ExitCode: 127}
	}

	cctx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()

	var stdinData []byte
	var out []byte
	for i, part := range parts {
		tokens, tokErr := tokenize(part)
		if tokErr != nil || len(tokens) == 0 {
			return ExecResult{Output: shellLikeUnsupported(strings.TrimSpace(part)), ExitCode: 127}
		}

		// Output redirection is honored on the final segment only.
		redirectPath := ""
		appendRedir := false
		if i == len(parts)-1 {
			clean := make([]string, 0, len(tokens))
			for j := 0; j < len(tokens); j++ {
				t := tokens[j]
				switch {
				case t == ">" || t == ">>":
					if j+1 >= len(tokens) {
						return ExecResult{Output: "ckad-simulator: syntax error near unexpected token `newline'\n", ExitCode: 2}
					}
					redirectPath = tokens[j+1]
					appendRedir = t == ">>"
					j++
				case len(t) > 2 && strings.HasPrefix(t, ">>"):
					redirectPath = t[2:]
					appendRedir = true
				case len(t) > 1 && strings.HasPrefix(t, ">"):
					redirectPath = t[1:]
				default:
					clean = append(clean, t)
				}
			}
			tokens = clean
		}

		bin, args := tokens[0], tokens[1:]
		if allowed, known := allowedPrefixes[bin]; !known || !allowed {
			msg := fmt.Sprintf("command not allowed in this simulator: %s\n", bin)
			if bin == "vi" || bin == "vim" || bin == "view" || bin == "nano" {
				msg = fmt.Sprintf("%s opens the built-in editor — run `%s <file>` in the exam terminal\n", bin, bin)
			}
			return ExecResult{Output: msg, ExitCode: 126}
		}
		if bin == "k" {
			// Common kubectl alias.
			bin = c.Binary
		}

		// Confine every argument that names a file on this host to the exam
		// sandbox, so `kubectl apply -f /pod.yaml` sees the editor-created
		// file while `cat /etc/passwd` and `rm ../../thing` cannot reach
		// the host filesystem.
		args, cerr := c.confineArgs(bin, args)
		if cerr != nil {
			return ExecResult{Output: fmt.Sprintf("ckad-simulator: %v\n", cerr), ExitCode: 1}
		}

		cmd := exec.CommandContext(cctx, bin, args...)
		cmd.Dir = c.homeDir
		if len(stdinData) > 0 {
			cmd.Stdin = bytes.NewReader(stdinData)
		}
		var err error
		out, err = cmd.CombinedOutput()

		if redirectPath != "" {
			full, rerr := c.ResolvePath(redirectPath)
			if rerr != nil {
				return ExecResult{Output: fmt.Sprintf("ckad-simulator: %v\n", rerr), ExitCode: 1}
			}
			flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
			if appendRedir {
				flags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
			}
			f, werr := os.OpenFile(full, flags, 0o644)
			if werr != nil {
				return ExecResult{Output: fmt.Sprintf("ckad-simulator: %v\n", werr), ExitCode: 1}
			}
			_, _ = f.Write(out)
			_ = f.Close()
			out = nil
		}

		if err != nil {
			res := ExecResult{Output: string(out)}
			var exitErr *exec.ExitError
			switch {
			case errors.As(err, &exitErr):
				res.ExitCode = exitErr.ExitCode()
			case errors.Is(err, exec.ErrNotFound):
				// Allow-listed, but the host does not have it installed.
				res.Output = fmt.Sprintf("%s: not installed in this exam environment\n", bin)
				res.ExitCode = 127
			default:
				res.Output = fmt.Sprintf("%s\n%s", res.Output, err.Error())
				res.ExitCode = 1
			}
			return truncateResult(res)
		}
		stdinData = out
	}

	return truncateResult(ExecResult{Output: string(out)})
}

func truncateResult(res ExecResult) ExecResult {
	if len(res.Output) > maxExecOutput {
		res.Output = res.Output[:maxExecOutput] + "\n…output truncated…"
	}
	return res
}

// splitTopLevel splits s on sep, ignoring separators inside quotes.
func splitTopLevel(s string, sep byte) []string {
	raw := make([]string, 0, 2)
	var cur strings.Builder
	quote := byte(0)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			cur.WriteByte(ch)
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch {
		case ch == '\'' || ch == '"':
			quote = ch
			cur.WriteByte(ch)
		case ch == sep:
			raw = append(raw, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(ch)
		}
	}
	raw = append(raw, cur.String())

	parts := make([]string, 0, len(raw))
	for _, p := range raw {
		if strings.TrimSpace(p) != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// tokenize splits a command segment into tokens, honoring single and double
// quotes. Remaining shell metacharacters are rejected — no real shell runs
// on the host.
func tokenize(segment string) ([]string, error) {
	var toks []string
	var cur strings.Builder
	inTok := false
	quote := byte(0)
	for i := 0; i < len(segment); i++ {
		ch := segment[i]
		switch {
		case quote == '\'':
			if ch == '\'' {
				quote = 0
			} else {
				cur.WriteByte(ch)
			}
		case quote == '"':
			if ch == '"' {
				quote = 0
			} else {
				cur.WriteByte(ch)
			}
		case ch == '\'' || ch == '"':
			quote = ch
			inTok = true
		case ch == ' ' || ch == '\t':
			if inTok {
				toks = append(toks, cur.String())
				cur.Reset()
				inTok = false
			}
		default:
			cur.WriteByte(ch)
			inTok = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	if inTok {
		toks = append(toks, cur.String())
	}
	for _, t := range toks {
		for _, meta := range []string{"&", ";", "`", "$("} {
			if strings.Contains(t, meta) {
				return nil, fmt.Errorf("unsupported shell syntax")
			}
		}
	}
	return toks, nil
}

// shellLikeUnsupported explains why a command could not run.
func shellLikeUnsupported(command string) string {
	for _, meta := range []string{"|", "&", ";", ">", "<", "`", "$("} {
		if strings.Contains(command, meta) {
			return fmt.Sprintf("ckad-simulator: shell pipes/redirection are not supported in this baseline terminal: %s\n", command)
		}
	}
	return fmt.Sprintf("ckad-simulator: command not found: %s\n", strings.Fields(command)[0])
}

// truncate limits output length for storage/transport.
func truncate(s string) string {
	if len(s) <= maxOutput {
		return s
	}
	return s[:maxOutput] + "…"
}
