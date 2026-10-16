package cmd

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Overridden only by the isolated pre-release integration build.
var csharpReleaseBase = "https://github.com/NotNull92/hera-agent-godot/releases/download"

func runCSharpHost(args []string) int {
	tag := version
	if len(args) == 0 || args[0] != "install" {
		fmt.Fprintln(os.Stderr, "usage: hera-godot csharp-host install [--version vX.Y.Z]")
		return 2
	}
	for i := 1; i < len(args); {
		if !isFlag(args[i], "--version") {
			fmt.Fprintln(os.Stderr, "unknown csharp-host install argument")
			return 2
		}
		value, next, err := takeFlagValue(args, i, "--version")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		tag, i = value, next
	}
	if !regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(tag) {
		fmt.Fprintln(os.Stderr, "specify --version vX.Y.Z (required for development builds)")
		return 2
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	deadline := requestTimeout
	if deadline == 0 {
		deadline = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	dest := filepath.Join(home, ".hera-agent-godot", "csharp")
	if err := installCSharpHost(ctx, http.DefaultClient, csharpReleaseBase+"/"+tag, dest); err != nil {
		fmt.Fprintln(os.Stderr, "csharp_host_install_failed:", err)
		return 1
	}
	return printData(&protocol.Response{OK: true, Data: map[string]any{"installed": true, "version": tag, "path": dest}})
}

func downloadHostAsset(ctx context.Context, client *http.Client, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download exceeds size limit")
	}
	return data, nil
}

func installCSharpHost(ctx context.Context, client *http.Client, base, dest string) error {
	checksums, err := downloadHostAsset(ctx, client, base+"/checksums.txt", 1<<20)
	if err != nil {
		return err
	}
	archive, err := downloadHostAsset(ctx, client, base+"/hera-csharp-host.zip", 64<<20)
	if err != nil {
		return err
	}
	want := ""
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == "hera-csharp-host.zip" {
			if want != "" {
				return fmt.Errorf("duplicate helper checksum")
			}
			want = fields[0]
		}
	}
	if want == "" || !strings.EqualFold(want, fmt.Sprintf("%x", sha256.Sum256(archive))) {
		return fmt.Errorf("helper SHA256 mismatch or missing checksum; installation unchanged")
	}
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return err
	}
	parent := filepath.Dir(dest)
	if err = os.MkdirAll(parent, 0700); err != nil {
		return err
	}
	lock := dest + ".install-lock"
	if err = os.Mkdir(lock, 0700); err != nil {
		return fmt.Errorf("installation locked: %w", err)
	}
	defer os.Remove(lock)
	stage, err := os.MkdirTemp(parent, ".csharp-stage-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	var total int64
	for _, entry := range zr.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := strings.TrimSuffix(entry.Name, "/")
		if !fs.ValidPath(name) || strings.ContainsAny(name, `\:`) || entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe helper archive entry %q", entry.Name)
		}
		target := filepath.Join(stage, filepath.FromSlash(name))
		if entry.FileInfo().IsDir() {
			if err = os.MkdirAll(target, 0700); err != nil {
				return err
			}
			continue
		}
		if !entry.Mode().IsRegular() {
			return fmt.Errorf("non-regular helper entry")
		}
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		src, err := entry.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			src.Close()
			return err
		}
		n, copyErr := io.Copy(out, io.LimitReader(src, (256<<20)-total+1))
		src.Close()
		closeErr := out.Close()
		total += n
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if total > 256<<20 {
			return fmt.Errorf("expanded helper exceeds size limit")
		}
	}
	for _, required := range []string{"Hera.CSharpHost.dll", "Hera.CSharpHost.deps.json", "Hera.CSharpHost.runtimeconfig.json", "Microsoft.CodeAnalysis.dll", "Microsoft.CodeAnalysis.CSharp.dll"} {
		info, err := os.Stat(filepath.Join(stage, required))
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("helper archive missing %s", required)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	backup := ""
	hadOld := false
	if info, statErr := os.Lstat(dest); statErr == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("destination is not a regular directory")
		}
		backup, err = os.MkdirTemp(parent, ".csharp-previous-")
		if err != nil {
			return err
		}
		if err = os.Remove(backup); err != nil {
			return err
		}
		if err = os.Rename(dest, backup); err != nil {
			return err
		}
		hadOld = true
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	if err = os.Rename(stage, dest); err != nil {
		if hadOld {
			if restore := os.Rename(backup, dest); restore != nil {
				return fmt.Errorf("%v; restore failed: %v; backup at %s", err, restore, backup)
			}
		}
		return err
	}
	if hadOld {
		if err = os.RemoveAll(backup); err != nil {
			return fmt.Errorf("installed but old backup cleanup failed at %s: %w", backup, err)
		}
	}
	return nil
}
