// Package provenance implements Specular's open Agent Provenance Protocol (v1).
//
// The protocol maps existing attestation.Provenance fields into a stable,
// agent-neutral envelope (schema specular.provenance/v1). It is a document
// format + emit/consume path — not a control plane. See
// docs/PRODUCT_INTENT.md §9 / P1 #3.
package provenance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/attestation"
)

// Schema identifies the Agent Provenance Protocol document format.
const Schema = "specular.provenance/v1"

// Version is the protocol major version carried inside the envelope.
const Version = "1"

// SessionsDir is the directory under .specular for session attestations.
const SessionsDir = "sessions"

// Document is the open, versioned provenance envelope. Fields map from
// attestation.Provenance (and the session id) without inventing new identity.
type Document struct {
	Schema   string    `json:"schema"`
	Version  string    `json:"version"`
	Session  string    `json:"session,omitempty"`
	Harness  string    `json:"harness,omitempty"`
	Governed bool      `json:"governed,omitempty"`
	Worktree *Worktree `json:"worktree,omitempty"`
	Git      *Git      `json:"git,omitempty"`
	// Source is the relative attestation path this document was projected from.
	Source string `json:"source,omitempty"`
}

// Worktree records managed Git worktree isolation when present.
type Worktree struct {
	Path   string `json:"path,omitempty"`
	Branch string `json:"branch,omitempty"`
	Name   string `json:"name,omitempty"`
}

// Git records repository coordinates from attestation provenance.
type Git struct {
	Repo   string `json:"repo,omitempty"`
	Commit string `json:"commit,omitempty"`
	Branch string `json:"branch,omitempty"`
	Dirty  bool   `json:"dirty,omitempty"`
}

// ProtocolRef is an additive gate/evidence pointer to the v1 protocol.
// Present when session attestation(s) exist; does not embed the full document.
type ProtocolRef struct {
	Schema   string   `json:"schema"`
	Version  string   `json:"version"`
	Sessions []string `json:"sessions,omitempty"`
}

// FromProvenance projects attestation.Provenance into a v1 Document.
func FromProvenance(sessionID string, p attestation.Provenance) *Document {
	doc := &Document{
		Schema:   Schema,
		Version:  Version,
		Session:  strings.TrimSpace(sessionID),
		Harness:  strings.TrimSpace(p.Harness),
		Governed: p.Governed,
	}
	if p.WorktreePath != "" || p.WorktreeBranch != "" || p.WorktreeName != "" {
		doc.Worktree = &Worktree{
			Path:   p.WorktreePath,
			Branch: p.WorktreeBranch,
			Name:   p.WorktreeName,
		}
	}
	if p.GitRepo != "" || p.GitCommit != "" || p.GitBranch != "" || p.GitDirty {
		doc.Git = &Git{
			Repo:   p.GitRepo,
			Commit: p.GitCommit,
			Branch: p.GitBranch,
			Dirty:  p.GitDirty,
		}
	}
	return doc
}

// FromAttestation projects a signed attestation into a v1 Document.
func FromAttestation(sessionID string, att *attestation.Attestation) *Document {
	if att == nil {
		return &Document{Schema: Schema, Version: Version, Session: strings.TrimSpace(sessionID)}
	}
	return FromProvenance(sessionID, att.Provenance)
}

// NewProtocolRef builds the additive gate/evidence protocol pointer.
func NewProtocolRef(sessions []string) *ProtocolRef {
	out := make([]string, len(sessions))
	copy(out, sessions)
	return &ProtocolRef{
		Schema:   Schema,
		Version:  Version,
		Sessions: out,
	}
}

// SessionsPath returns .specular/sessions under root.
func SessionsPath(root string) string {
	return filepath.Join(root, ".specular", SessionsDir)
}

// LoadSession reads a session attestation and projects it to a v1 Document.
func LoadSession(root, sessionID string) (*Document, error) {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return nil, fmt.Errorf("provenance: empty session id")
	}
	rel := filepath.Join(".specular", SessionsDir, id+".attestation.json")
	path := filepath.Join(root, rel)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("provenance: load session %s: %w", id, err)
	}
	att, parseErr := attestation.FromJSON(raw)
	if parseErr != nil {
		return nil, fmt.Errorf("provenance: parse session %s: %w", id, parseErr)
	}
	doc := FromAttestation(id, att)
	doc.Source = filepath.ToSlash(rel)
	return doc, nil
}

// LoadLatest reads the newest session attestation by file modtime and
// projects it to a v1 Document.
func LoadLatest(root string) (*Document, error) {
	dir := SessionsPath(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("provenance: no session attestations (run specular session attest): %w", err)
	}
	var bestName string
	var bestMod time.Time
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".attestation.json") {
			continue
		}
		info, infoErr := e.Info()
		if infoErr != nil {
			continue
		}
		mod := info.ModTime()
		if bestName == "" || mod.After(bestMod) || (mod.Equal(bestMod) && name > bestName) {
			bestName = name
			bestMod = mod
		}
	}
	if bestName == "" {
		return nil, fmt.Errorf("provenance: no session attestations found under %s", dir)
	}
	id := strings.TrimSuffix(bestName, ".attestation.json")
	return LoadSession(root, id)
}

// ToJSON serializes the document with indentation.
func (d *Document) ToJSON() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

// FormatHuman renders an auditor-facing provenance summary.
func FormatHuman(d *Document) string {
	if d == nil {
		return "No provenance document.\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Agent Provenance (%s)\n", d.Schema)
	fmt.Fprintf(&b, "Version      %s\n", d.Version)
	if d.Session != "" {
		fmt.Fprintf(&b, "Session      %s\n", d.Session)
	}
	if d.Harness != "" {
		fmt.Fprintf(&b, "Harness      %s\n", d.Harness)
	}
	fmt.Fprintf(&b, "Governed     %v\n", d.Governed)
	if line := formatWorktreeLine(d.Worktree); line != "" {
		fmt.Fprintf(&b, "Worktree     %s\n", line)
	}
	if line := formatGitLine(d.Git); line != "" {
		fmt.Fprintf(&b, "Git          %s\n", line)
	}
	if d.Source != "" {
		fmt.Fprintf(&b, "Source       %s\n", d.Source)
	}
	writeProvenanceRefs(&b, d.Session, "verify")
	return b.String()
}

// writeProvenanceRefs jumps APP documents to session / explain / verify|show
// (FormatExplain session Refs parity).
func writeProvenanceRefs(b *strings.Builder, sessionID, peer string) {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return
	}
	b.WriteString("Refs\n")
	fmt.Fprintf(b, "Session      specular session show %s\n", sid)
	fmt.Fprintf(b, "Explain      specular explain --session %s\n", sid)
	switch peer {
	case "show":
		fmt.Fprintf(b, "Show         specular provenance show %s\n", sid)
	default:
		fmt.Fprintf(b, "Verify       specular provenance verify %s\n", sid)
	}
}

func formatWorktreeLine(wt *Worktree) string {
	if wt == nil {
		return ""
	}
	line := wt.Path
	if wt.Branch != "" {
		if line != "" {
			line = fmt.Sprintf("%s (%s)", line, wt.Branch)
		} else {
			line = wt.Branch
		}
	}
	if wt.Name == "" {
		return line
	}
	if line != "" {
		return fmt.Sprintf("%s [%s]", line, wt.Name)
	}
	return wt.Name
}

func formatGitLine(g *Git) string {
	if g == nil {
		return ""
	}
	parts := make([]string, 0, 3)
	if g.Commit != "" {
		parts = append(parts, shortSHA(g.Commit))
	}
	if g.Branch != "" {
		parts = append(parts, "on "+g.Branch)
	}
	if g.Repo != "" && len(parts) == 0 {
		parts = append(parts, g.Repo)
	}
	line := strings.Join(parts, " ")
	if !g.Dirty {
		return line
	}
	if line != "" {
		return line + " (dirty)"
	}
	return "dirty"
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
