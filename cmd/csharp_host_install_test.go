package cmd

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCSharpHost(t *testing.T) {
	for _, scenario := range []string{"install", "mismatch", "missing-checksum", "traversal", "missing-file"} {
		t.Run(scenario, func(t *testing.T) {
			var archive bytes.Buffer
			zw := zip.NewWriter(&archive)
			names := []string{"Hera.CSharpHost.dll", "Hera.CSharpHost.deps.json", "Hera.CSharpHost.runtimeconfig.json", "Microsoft.CodeAnalysis.dll", "Microsoft.CodeAnalysis.CSharp.dll", "ko/resources.dll"}
			if scenario == "traversal" {
				names = append(names, "../outside")
			}
			if scenario == "missing-file" {
				names = names[1:]
			}
			for _, name := range names {
				w, err := zw.Create(name)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := w.Write([]byte("new")); err != nil {
					t.Fatal(err)
				}
			}
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}
			sum := fmt.Sprintf("%x  hera-csharp-host.zip\n", sha256.Sum256(archive.Bytes()))
			if scenario == "mismatch" {
				sum = strings.Repeat("0", 64) + "  hera-csharp-host.zip\n"
			}
			if scenario == "missing-checksum" {
				sum = ""
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v2.0.0/checksums.txt":
					fmt.Fprint(w, sum)
				case "/v2.0.0/hera-csharp-host.zip":
					w.Write(archive.Bytes())
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			root := t.TempDir()
			dest := filepath.Join(root, "csharp")
			if err := os.Mkdir(dest, 0700); err != nil {
				t.Fatal(err)
			}
			old := filepath.Join(dest, "old-install")
			if err := os.WriteFile(old, []byte("old"), 0600); err != nil {
				t.Fatal(err)
			}
			err := installCSharpHost(context.Background(), server.Client(), server.URL+"/v2.0.0", dest)
			if scenario == "install" {
				if err != nil {
					t.Fatal(err)
				}
				if data, err := os.ReadFile(filepath.Join(dest, "Hera.CSharpHost.dll")); err != nil || string(data) != "new" {
					t.Fatalf("bad installed DLL: %q %v", data, err)
				}
				if _, err := os.Stat(old); !os.IsNotExist(err) {
					t.Fatal("stale install retained")
				}
			} else {
				if err == nil {
					t.Fatal("unsafe installation succeeded")
				}
				if data, err := os.ReadFile(old); err != nil || string(data) != "old" {
					t.Fatal("existing install changed")
				}
			}
			if _, err := os.Stat(filepath.Join(root, "outside")); !os.IsNotExist(err) {
				t.Fatal("archive escaped destination")
			}
		})
	}
}
