// Package bundle handles the directory layout of a Reunion family file bundle.
package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Bundle represents the directory structure of a Reunion family file.
type Bundle struct {
	Path       string
	Signature  string // path to familyfile.signature
	FamilyData string // path to familyfile.familydata (or a legacy *.familyfile)
	Caches     map[string]string // cache name -> path
	Members    []MemberDir
	NoteFiles  []string // all .note file paths across all members
}

// MemberDir represents a .member directory.
type MemberDir struct {
	Name       string
	Path       string
	Changes    string // path to .changes file, if any
	NotesDir   string // path to .notes directory, if any
	MediaDir   string // path to .media directory, if any
	NoteFiles  []string
}

// OpenBundle validates and inventories a Reunion bundle directory.
func OpenBundle(path string) (*Bundle, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access bundle: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("bundle path is not a directory: %s", path)
	}

	b := &Bundle{
		Path:   path,
		Caches: make(map[string]string),
	}

	// Check for required files
	sigPath := filepath.Join(path, "familyfile.signature")
	if _, err := os.Stat(sigPath); err == nil {
		b.Signature = sigPath
	}

	fdPath := filepath.Join(path, "familyfile.familydata")
	if _, err := os.Stat(fdPath); err == nil {
		b.FamilyData = fdPath
	}

	// Discover cache files
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading bundle directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		// Reunion 10 stores the data file as "Family File.familyfile".
		if b.FamilyData == "" && !entry.IsDir() && strings.HasSuffix(name, ".familyfile") {
			b.FamilyData = filepath.Join(path, name)
		}
		if strings.HasSuffix(name, ".cache") {
			b.Caches[name] = filepath.Join(path, name)
		}
		if entry.IsDir() && strings.HasSuffix(name, ".member") {
			md, err := inventoryMember(filepath.Join(path, name), name)
			if err != nil {
				continue
			}
			b.Members = append(b.Members, md)
			b.NoteFiles = append(b.NoteFiles, md.NoteFiles...)
		}
	}

	return b, nil
}

func inventoryMember(dirPath, name string) (MemberDir, error) {
	md := MemberDir{
		Name: strings.TrimSuffix(name, ".member"),
		Path: dirPath,
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return md, err
	}

	for _, entry := range entries {
		eName := entry.Name()
		ePath := filepath.Join(dirPath, eName)
		if strings.HasSuffix(eName, ".changes") {
			md.Changes = ePath
		}
		if strings.HasSuffix(eName, ".notes") && entry.IsDir() {
			md.NotesDir = ePath
			noteEntries, err := os.ReadDir(ePath)
			if err == nil {
				for _, ne := range noteEntries {
					if strings.HasSuffix(ne.Name(), ".note") {
						notePath := filepath.Join(ePath, ne.Name())
						md.NoteFiles = append(md.NoteFiles, notePath)
					}
				}
			}
		}
		if strings.HasSuffix(eName, ".media") && entry.IsDir() {
			md.MediaDir = ePath
		}
	}

	return md, nil
}
