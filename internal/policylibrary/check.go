package policylibrary

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ArtifactStatus is one control→evidence presence check.
type ArtifactStatus struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Why     string `json:"why,omitempty"`
	Present bool   `json:"present"`
	Matches int    `json:"matches,omitempty"`
	Note    string `json:"note,omitempty"`
}

// CheckReport is an auditor-facing control→evidence status for one pack.
type CheckReport struct {
	ID         string           `json:"id"`
	Framework  string           `json:"framework"`
	Control    string           `json:"control"`
	Title      string           `json:"title"`
	Root       string           `json:"root"`
	OK         bool             `json:"ok"`
	Present    int              `json:"present"`
	Missing    int              `json:"missing"`
	Artifacts  []ArtifactStatus `json:"artifacts"`
	Disclaimer string           `json:"disclaimer"`
}

const checkDisclaimer = "Implements Specular controls and evidence paths; does not certify organizational compliance."

// Check evaluates whether mapped artifacts exist under root for the pack id.
func Check(root, id string) (CheckReport, error) {
	e, err := Get(id)
	if err != nil {
		return CheckReport{}, err
	}
	if root == "" {
		root = "."
	}
	abs, absErr := filepath.Abs(root)
	if absErr != nil {
		return CheckReport{}, fmt.Errorf("policylibrary: resolve root: %w", absErr)
	}
	rep := CheckReport{
		ID:         e.ID,
		Framework:  e.Framework,
		Control:    e.Control,
		Title:      e.Title,
		Root:       abs,
		Disclaimer: checkDisclaimer,
	}
	for _, a := range e.Artifacts {
		st := checkArtifact(abs, a)
		rep.Artifacts = append(rep.Artifacts, st)
		if st.Present {
			rep.Present++
		} else {
			rep.Missing++
		}
	}
	rep.OK = rep.Missing == 0 && len(rep.Artifacts) > 0
	return rep, nil
}

func checkArtifact(root string, a Artifact) ArtifactStatus {
	st := ArtifactStatus{Kind: a.Kind, Path: a.Path, Why: a.Why}
	pattern := strings.TrimSpace(a.Path)
	if pattern == "" {
		st.Note = "empty path"
		return st
	}
	// Absolute patterns are checked as-is; relative join under root.
	full := pattern
	if !filepath.IsAbs(pattern) {
		full = filepath.Join(root, filepath.FromSlash(pattern))
	}

	if strings.ContainsAny(pattern, "*?[") {
		matches, err := filepath.Glob(full)
		if err != nil {
			st.Note = err.Error()
			return st
		}
		st.Matches = len(matches)
		st.Present = len(matches) > 0
		if !st.Present {
			st.Note = "no matches"
		}
		return st
	}

	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			st.Note = "missing"
			return st
		}
		st.Note = err.Error()
		return st
	}
	st.Present = true
	st.Matches = 1
	if info.IsDir() {
		st.Note = "directory present"
	} else {
		st.Note = "file present"
	}
	return st
}

// FormatCheckHuman renders an auditor-facing check board.
func FormatCheckHuman(rep CheckReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "CONTROL PACK CHECK\n")
	fmt.Fprintf(&b, "──────────────────────────────────────────────\n")
	fmt.Fprintf(&b, "Pack         %s\n", rep.ID)
	fmt.Fprintf(&b, "Framework    %s\n", rep.Framework)
	fmt.Fprintf(&b, "Control      %s\n", rep.Control)
	fmt.Fprintf(&b, "Title        %s\n", rep.Title)
	fmt.Fprintf(&b, "Root         %s\n", rep.Root)
	fmt.Fprintf(&b, "Status       %s  (present=%d missing=%d)\n",
		checkStatusWord(rep.OK), rep.Present, rep.Missing)
	b.WriteString("Artifacts\n")
	for _, a := range rep.Artifacts {
		mark := "✗"
		if a.Present {
			mark = "✓"
		}
		fmt.Fprintf(&b, "  %s %s → %s\n", mark, a.Kind, a.Path)
		if a.Why != "" {
			fmt.Fprintf(&b, "      why: %s\n", a.Why)
		}
		if a.Note != "" {
			fmt.Fprintf(&b, "      %s", a.Note)
			if a.Matches > 0 {
				fmt.Fprintf(&b, " (matches=%d)", a.Matches)
			}
			b.WriteString("\n")
		}
	}
	fmt.Fprintf(&b, "──────────────────────────────────────────────\n")
	fmt.Fprintf(&b, "Note: %s\n", rep.Disclaimer)
	return b.String()
}

func checkStatusWord(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}
