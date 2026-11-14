package compress

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"filippo.io/age"
	"github.com/klauspost/compress/zstd"
)

func TestPasswordProtectedBackups(t *testing.T) {
	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "first.txt")
	second := filepath.Join(sourceDir, "second.txt")
	contents := bytes.Repeat([]byte("backup contents\n"), 8192)
	for _, path := range []string{source, second} {
		if err := os.WriteFile(path, contents, 0600); err != nil {
			t.Fatal(err)
		}
	}
	const password = " test # passphrase "
	identity, err := age.NewScryptIdentity(password)
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"zstd", "gzip", "none"} {
		for _, kind := range []string{"file", "files", "dir"} {
			t.Run(method+"/"+kind, func(t *testing.T) {
				archive := kind != "file"
				output := filepath.Join(t.TempDir(), "backup"+Extension(method, archive, true))
				var err error
				switch kind {
				case "file":
					err = CompressFile(source, output, method, password)
				case "files":
					err = CompressFiles([]string{source, second}, output, method, password)
				case "dir":
					err = CompressDir(sourceDir, output, []string{"second.txt"}, method, password)
				}
				if err != nil {
					t.Fatal(err)
				}
				info, err := os.Stat(output)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm() != 0600 {
					t.Fatalf("permissions = %o, want 600", info.Mode().Perm())
				}
				f, err := os.Open(output)
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				decrypted, err := age.Decrypt(f, identity)
				if err != nil {
					t.Fatal(err)
				}
				clear, err := io.ReadAll(decrypted)
				if err != nil {
					t.Fatal(err)
				}
				payload := readPayload(t, method, clear)
				if !archive {
					if !bytes.Equal(payload, contents) {
						t.Fatal("restored file differs from source")
					}
					return
				}

				var names []string
				reader := tar.NewReader(bytes.NewReader(payload))
				for {
					header, err := reader.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
					if header.Typeflag == tar.TypeDir {
						continue
					}
					entry, err := io.ReadAll(reader)
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(entry, contents) {
						t.Fatalf("restored %s differs from source", header.Name)
					}
					names = append(names, header.Name)
				}
				want := []string{"first.txt"}
				if kind == "files" {
					want = append(want, "second.txt")
				}
				if !slices.Equal(names, want) {
					t.Fatalf("restored names = %v, want %v", names, want)
				}
			})
		}
	}
}

func TestArchiveFailureRemovesPartialOutput(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "backup.tar.age")
	err := CompressFiles([]string{filepath.Join(dir, "missing")}, output, "none", "test password")
	if err == nil {
		t.Fatal("expected missing source to fail")
	}
	if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial backup was not removed: %v", err)
	}
}

func TestAgeRejectsWrongPasswordAndTampering(t *testing.T) {
	output := filepath.Join(t.TempDir(), "backup.age")
	if err := writeBackup(output, "none", "correct password", func(w io.Writer) error {
		_, err := io.WriteString(w, "data")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	ciphertext, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	wrong, err := age.NewScryptIdentity("wrong password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := age.Decrypt(bytes.NewReader(ciphertext), wrong); err == nil {
		t.Fatal("wrong password accepted")
	}
	correct, err := age.NewScryptIdentity("correct password")
	if err != nil {
		t.Fatal(err)
	}
	ciphertext[len(ciphertext)-1] ^= 1
	decrypted, err := age.Decrypt(bytes.NewReader(ciphertext), correct)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(decrypted); err == nil {
		t.Fatal("modified ciphertext accepted")
	}
}

func readPayload(t *testing.T, method string, data []byte) []byte {
	t.Helper()
	switch method {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		payload, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		return payload
	case "zstd":
		reader, err := zstd.NewReader(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		payload, err := reader.DecodeAll(data, nil)
		if err != nil {
			t.Fatal(err)
		}
		return payload
	default:
		return data
	}
}
