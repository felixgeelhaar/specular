package main

import (
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "os"
    "path/filepath"
    "strings"
)

func main() {
    var files []string
    err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() {
            // Skip vendor, .git, testdata, and hidden dirs
            base := filepath.Base(path)
            if base == "vendor" || base == ".git" || strings.HasPrefix(base, ".") || base == "testdata" {
                return filepath.SkipDir
            }
            return nil
        }
        if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
            files = append(files, path)
        }
        return nil
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
        os.Exit(1)
    }

    violations := false
    for _, file := range files {
        src, err := os.ReadFile(file)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
            os.Exit(1)
        }
        fset := token.NewFileSet()
        f, err := parser.ParseFile(fset, file, src, parser.AllErrors)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Parse error in %s: %v\n", file, err)
            continue
        }
        // Direct exec.Command* usage
        ast.Inspect(f, func(n ast.Node) bool {
            call, ok := n.(*ast.CallExpr)
            if !ok {
                return true
            }
            sel, ok := call.Fun.(*ast.SelectorExpr)
            if ok {
                if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "exec" && strings.HasPrefix(sel.Sel.Name, "Command") {
                    report(file, fset.Position(call.Pos()), "direct exec.Command usage – use safeutil.SafeCommand")
                    violations = true
                }
            }
            // File write with permissive mode (>=0644)
            if funIdent, ok := call.Fun.(*ast.Ident); ok && (funIdent.Name == "WriteFile" || funIdent.Name == "WriteFile" || funIdent.Name == "WriteFile") {
                // Simplified: any WriteFile call considered a violation (requires mode check for real enforcement)
                report(file, fset.Position(call.Pos()), "os.WriteFile used without safe mode – ensure permissions are restrictive")
                violations = true
            }
            return true
        })
    }
    if violations {
        fmt.Fprintln(os.Stderr, "Guardrail violations detected. Failing build.")
        os.Exit(1)
    }
    fmt.Println("Guardrail passed – no insecure exec or file-write patterns found.")
}

func report(file string, pos token.Position, msg string) {
    fmt.Printf("%s:%d:%d: %s\n", file, pos.Line, pos.Column, msg)
    // Provide quick fix suggestion if possible
    if strings.Contains(msg, "exec.Command") {
        fmt.Printf("  Suggestion: replace with safeutil.SafeCommand(context.Background(), ...)\n")
    }
    if strings.Contains(msg, "WriteFile") {
        fmt.Printf("  Suggestion: validate file mode (e.g., 0600) or use safe write utilities\n")
    }
}
