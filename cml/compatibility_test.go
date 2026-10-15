package cml

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type legacyDiagnostic struct {
	Severity Severity
	Pos      Position
	Message  string
}
type compatibilityEntry struct {
	File          string
	Error         string             `json:",omitempty"`
	PrintedSHA256 string             `json:",omitempty"`
	Diagnostics   []legacyDiagnostic `json:",omitempty"`
}

// Pins pre-foundation behavior independently of additive diagnostic metadata.
// The fixture was captured before production edits, not from the new loader.
func TestCompatibilityBaseline(t *testing.T) {
	var got []compatibilityEntry
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(cwd)
	for _, dir := range []string{"conformance", "corpus"} {
		err := filepath.WalkDir(filepath.Join("..", "testdata", dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".cml") {
				return nil
			}
			rel, err := filepath.Rel(filepath.Join("..", "testdata"), path)
			if err != nil {
				return err
			}
			item := compatibilityEntry{File: filepath.ToSlash(rel)}
			var model *Model
			if dir == "corpus" {
				model, err = ParseFile(path)
			} else {
				var src []byte
				src, err = os.ReadFile(path)
				if err == nil {
					model, err = Parse(path, src)
				}
			}
			if err != nil {
				item.Error = strings.ReplaceAll(filepath.ToSlash(err.Error()), filepath.ToSlash(root)+"/", "")
			} else {
				sum := sha256.Sum256([]byte(model.String()))
				item.PrintedSHA256 = hex.EncodeToString(sum[:])
				for _, d := range model.Validate() {
					pos := d.Pos
					if filepath.IsAbs(pos.File) {
						pos.File, err = filepath.Rel(root, pos.File)
						if err != nil {
							return err
						}
					} else {
						pos.File = strings.TrimPrefix(filepath.ToSlash(pos.File), "../")
					}
					pos.File = filepath.ToSlash(pos.File)
					item.Diagnostics = append(item.Diagnostics, legacyDiagnostic{d.Severity, pos, d.Message})
				}
			}
			got = append(got, item)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(got) != 328 {
		t.Fatalf("corpus entries=%d, want 328", len(got))
	}
	fixture := filepath.Join("..", "testdata", "compatibility", "baseline.json")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var want []compatibilityEntry
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	if len(want) != len(got) {
		t.Fatalf("baseline entries: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Errorf("compatibility changed for %s\ngot: %+v\nwant: %+v", got[i].File, got[i], want[i])
		}
	}
}
