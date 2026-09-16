package checker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
)

// newTestChecker returns a Checker sandboxed to a throwaway directory.
func newTestChecker(t *testing.T) *Checker {
	t.Helper()
	home := t.TempDir()
	return &Checker{Binary: DefaultBinary, Timeout: DefaultTimeout, homeDir: home}
}

// TestConfineArgsBlocksEscapes covers the arguments that must never reach
// the host filesystem.
func TestConfineArgsBlocksEscapes(t *testing.T) {
	c := newTestChecker(t)

	rejected := []struct {
		name string
		bin  string
		args []string
	}{
		{"relative traversal", "cat", []string{"../../../../etc/passwd"}},
		{"traversal mid-path", "cat", []string{"sub/../../../../etc/passwd"}},
		{"parent directory listing", "ls", []string{".."}},
		{"delete outside sandbox", "rm", []string{"-rf", "../../thing"}},
		{"copy out of sandbox", "cp", []string{"secret.txt", "../../../tmp/stolen"}},
		{"flag value escapes", "grep", []string{"--file=../../../etc/passwd", "x"}},
		{"kubectl manifest escapes", "kubectl", []string{"apply", "-f", "../../../etc/passwd"}},
		{"kubectl from-file escapes", "kubectl", []string{"create", "secret", "generic", "s", "--from-file=../../../etc/shadow"}},
		{"kubectl keyed from-file escapes", "kubectl", []string{"create", "cm", "c", "--from-file=key=../../../etc/shadow"}},
		{"kubectl kubeconfig escapes", "kubectl", []string{"--kubeconfig", "../../../home/user/.kube/config", "get", "pods"}},
		{"kubectl cp out of sandbox", "kubectl", []string{"cp", "pod:/etc/passwd", "../../../tmp/stolen"}},
		{"awk script file escapes", "awk", []string{"-f", "../../../etc/passwd"}},
		{"sed explicit traversal operand", "sed", []string{"../../../etc/passwd"}},
		{"file url", "curl", []string{"file:///etc/passwd"}},
	}

	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := c.confineArgs(tc.bin, tc.args); err == nil {
				t.Fatalf("expected %s %v to be rejected, but it was allowed", tc.bin, tc.args)
			}
		})
	}
}

// TestConfineArgsRerootsAbsolutePaths checks that absolute paths are
// re-rooted into the sandbox rather than reaching the host, matching how
// the built-in editors treat them.
func TestConfineArgsRerootsAbsolutePaths(t *testing.T) {
	c := newTestChecker(t)

	cases := []struct {
		name string
		bin  string
		args []string
		want []string
	}{
		{"host file read", "cat", []string{"/etc/passwd"},
			[]string{filepath.Join(c.homeDir, "etc/passwd")}},
		{"recursive delete", "rm", []string{"-rf", "/etc"},
			[]string{"-rf", filepath.Join(c.homeDir, "etc")}},
		{"editor-created manifest", "kubectl", []string{"apply", "-f", "/pod.yaml"},
			[]string{"apply", "-f", filepath.Join(c.homeDir, "pod.yaml")}},
		{"find from root", "find", []string{"/", "-name", "*.yaml"},
			[]string{c.homeDir, "-name", "*.yaml"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := c.confineArgs(tc.bin, tc.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Join(got, " ") != strings.Join(tc.want, " ") {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestConfineArgsPreservesNonPathArguments guards the existing terminal
// features: object names, JSONPath, patterns, URLs and paths that refer to
// a pod's filesystem must all survive untouched.
func TestConfineArgsPreservesNonPathArguments(t *testing.T) {
	c := newTestChecker(t)

	unchanged := []struct {
		name string
		bin  string
		args []string
	}{
		{"jsonpath", "kubectl", []string{"get", "pods", "-o", "jsonpath={.items[0].metadata.name}"}},
		{"namespace flag", "kubectl", []string{"get", "pods", "-n", "q34"}},
		{"exec into pod reads pod paths", "kubectl", []string{"exec", "web", "--", "cat", "/etc/nginx/nginx.conf"}},
		{"exec shell", "kubectl", []string{"exec", "-it", "web", "--", "/bin/sh"}},
		{"apply from stdin", "kubectl", []string{"apply", "-f", "-"}},
		{"image with registry path", "kubectl", []string{"run", "web", "--image=docker.io/library/nginx:1.25"}},
		{"cp remote to bare name", "kubectl", []string{"cp", "pod:/etc/hosts", "hosts.txt"}},
		{"grep pattern with slash", "grep", []string{"namespace/q34"}},
		{"sed substitution", "sed", []string{"s/nginx/apache/", "pod.yaml"}},
		{"awk program", "awk", []string{"{print $1}", "out.txt"}},
		{"echo with slashes", "echo", []string{"a/b/c"}},
		{"basename is pure text", "basename", []string{"/usr/local/bin/kubectl"}},
		{"http url", "curl", []string{"http://10.0.0.1/healthz"}},
		{"bare filenames", "cat", []string{"pod.yaml"}},
	}

	for _, tc := range unchanged {
		t.Run(tc.name, func(t *testing.T) {
			got, err := c.confineArgs(tc.bin, tc.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Join(got, " ") != strings.Join(tc.args, " ") {
				t.Fatalf("arguments were rewritten:\n got %v\nwant %v", got, tc.args)
			}
		})
	}
}

// TestConfineArgsRelativePathsStayInsideSandbox checks that ordinary
// relative paths still work and resolve within the exam home.
func TestConfineArgsRelativePathsStayInsideSandbox(t *testing.T) {
	c := newTestChecker(t)
	got, err := c.confineArgs("cat", []string{"manifests/pod.yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(c.homeDir, "manifests/pod.yaml")
	if got[0] != want {
		t.Fatalf("got %q, want %q", got[0], want)
	}
}

// TestExecCannotReadHostFiles is the end-to-end form of the escape: the
// command really runs, and must not return host content.
func TestExecCannotReadHostFiles(t *testing.T) {
	c := newTestChecker(t)
	for _, cmd := range []string{
		"cat /etc/passwd",
		"cat ../../../../etc/passwd",
		"ls ../..",
	} {
		res := c.Exec(context.Background(), cmd)
		if strings.Contains(res.Output, "root:x:0:0") {
			t.Fatalf("%q leaked host /etc/passwd: %q", cmd, res.Output)
		}
		if res.ExitCode == 0 && strings.Contains(cmd, "passwd") {
			t.Fatalf("%q unexpectedly succeeded: %q", cmd, res.Output)
		}
	}
}

// TestExecCannotDeleteHostFiles proves the write side of the sandbox.
func TestExecCannotDeleteHostFiles(t *testing.T) {
	c := newTestChecker(t)
	outside := filepath.Join(t.TempDir(), "canary.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	c.Exec(context.Background(), "rm -f "+outside)
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("file outside the sandbox was deleted: %v", err)
	}

	c.Exec(context.Background(), "rm -rf ../../"+filepath.Base(filepath.Dir(outside)))
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("file outside the sandbox was deleted via traversal: %v", err)
	}
}

// TestExecPreservesTerminalFeatures covers the shell behaviour candidates
// rely on: pipes, redirection, the sandbox cwd and the editor hint.
func TestExecPreservesTerminalFeatures(t *testing.T) {
	c := newTestChecker(t)
	ctx := context.Background()

	if res := c.Exec(ctx, "echo hello > probe.txt"); res.ExitCode != 0 {
		t.Fatalf("redirection failed: %+v", res)
	}
	if res := c.Exec(ctx, "cat probe.txt"); !strings.Contains(res.Output, "hello") {
		t.Fatalf("could not read back redirected file: %+v", res)
	}
	if res := c.Exec(ctx, "echo more >> probe.txt"); res.ExitCode != 0 {
		t.Fatalf("append redirection failed: %+v", res)
	}
	if res := c.Exec(ctx, "cat probe.txt"); !strings.Contains(res.Output, "hello") ||
		!strings.Contains(res.Output, "more") {
		t.Fatalf("append lost content: %+v", res)
	}
	if res := c.Exec(ctx, "echo one/two/three | grep two/three"); !strings.Contains(res.Output, "one/two/three") {
		t.Fatalf("pipe with a slash in the pattern failed: %+v", res)
	}
	if res := c.Exec(ctx, "vi pod.yaml"); !strings.Contains(res.Output, "built-in editor") {
		t.Fatalf("editor hint lost: %+v", res)
	}
	if res := c.Exec(ctx, "python3 -c print(1)"); res.ExitCode != 126 {
		t.Fatalf("binary outside the allow-list should be refused: %+v", res)
	}
	if res := c.Exec(ctx, "curl http://example.com; rm -rf /"); res.ExitCode != 127 {
		t.Fatalf("shell metacharacters should be refused: %+v", res)
	}

	// Files written through the editor are visible to terminal commands.
	if err := c.WriteFile("/manifests/pod.yaml", "kind: Pod\n"); err != nil {
		t.Fatal(err)
	}
	if res := c.Exec(ctx, "cat /manifests/pod.yaml"); !strings.Contains(res.Output, "kind: Pod") {
		t.Fatalf("editor-created file not readable from the terminal: %+v", res)
	}
}

// TestPrepareWritesProvidedFiles covers the setup step that hands the
// candidate a manifest to inspect or repair: it must land in the sandbox
// and be readable through the terminal at the path the task names.
func TestPrepareWritesProvidedFiles(t *testing.T) {
	c := newTestChecker(t)
	const body = "apiVersion: v1\nkind: Deployment\n"

	q := &models.Question{
		ID: "tq-08",
		Prepare: []models.SetupStep{
			{Name: "place broken manifest", File: "/root/broken-deploy.yaml", FileContent: body},
		},
	}
	logs := c.Prepare(context.Background(), []*models.Question{q})
	if len(logs) != 1 || !strings.Contains(logs[0], "ok") {
		t.Fatalf("prepare did not report success: %v", logs)
	}

	got, err := c.ReadFile("/root/broken-deploy.yaml")
	if err != nil {
		t.Fatalf("file was not created: %v", err)
	}
	if got != body {
		t.Fatalf("content mismatch:\n got %q\nwant %q", got, body)
	}

	// And the candidate can reach it from the terminal.
	if res := c.Exec(context.Background(), "cat /root/broken-deploy.yaml"); !strings.Contains(res.Output, "kind: Deployment") {
		t.Fatalf("terminal cannot read the provided file: %+v", res)
	}
}

// TestPrepareFileStepCannotEscapeSandbox makes sure a malformed question
// cannot write outside the exam home.
func TestPrepareFileStepCannotEscapeSandbox(t *testing.T) {
	c := newTestChecker(t)
	q := &models.Question{
		ID:      "bad",
		Prepare: []models.SetupStep{{Name: "escape", File: "../../../../tmp/pwned-by-seed.txt", FileContent: "x"}},
	}
	logs := c.Prepare(context.Background(), []*models.Question{q})
	if len(logs) != 1 || !strings.Contains(logs[0], "failed") {
		t.Fatalf("expected the escaping file step to fail: %v", logs)
	}
	if _, err := os.Stat("/tmp/pwned-by-seed.txt"); err == nil {
		t.Fatal("a prepare step wrote outside the sandbox")
	}
}

// TestResolvePathRejectsEscapes guards the shared path helper.
func TestResolvePathRejectsEscapes(t *testing.T) {
	c := newTestChecker(t)
	if _, err := c.ResolvePath("../../etc/passwd"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
	got, err := c.ResolvePath("/etc/passwd")
	if err != nil {
		t.Fatalf("absolute paths should be re-rooted, got error: %v", err)
	}
	if want := filepath.Join(c.homeDir, "etc/passwd"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
