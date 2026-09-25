package ielts_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

// Command-line entry points behind `api validate-test` and `api import-test`
// (cmd/api/main.go): how a test body written outside the admin UI — by the
// YouPass script or the ielts-reading-import Claude Code skill — is checked
// against exactly the rules POST /api/tests applies, and then loaded.

// RunValidateCmd validates one CreateTestRequest JSON file ("-" reads
// stdin) and prints either a one-line summary or the first problem found.
// It needs no database or config. Exit codes: 0 valid, 1 invalid, 2 usage.
func RunValidateCmd(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: api validate-test FILE|-")
		return 2
	}
	req, err := readTestRequest(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := req.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid: %v\n", err)
		return 1
	}
	fmt.Println("valid: " + summarizeTest(req))
	return 0
}

// RunImportCmd validates a CreateTestRequest JSON file ("-" reads stdin)
// and inserts it, or with --replace ID overwrites that existing test of the
// same skill (for re-importing corrected content). Beware replacing a test
// people have already taken with renumbered questions: their stored
// answers are keyed by question number.
func RunImportCmd(db *sql.DB, args []string) int {
	fs := flag.NewFlagSet("import-test", flag.ContinueOnError)
	replace := fs.Uint64("replace", 0, "overwrite this existing test id instead of creating a new test")
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: api import-test [--replace ID] FILE|-")
		return 2
	}
	req, err := readTestRequest(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := req.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid: %v\n", err)
		return 1
	}

	ctx := context.Background()
	repo := NewRepository(db)
	t := req.Test()
	if *replace != 0 {
		existing, err := repo.GetTestByID(ctx, *replace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "test %d: %v\n", *replace, err)
			return 1
		}
		if existing.Skill != t.Skill {
			fmt.Fprintf(os.Stderr, "test %d is a %s test, not %s\n", *replace, existing.Skill, t.Skill)
			return 1
		}
		t.ID = *replace
		if err := repo.ReplaceTest(ctx, &t); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("replaced test %d: %s\n", t.ID, summarizeTest(req))
		return 0
	}
	created, err := repo.CreateTest(ctx, &t)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("created test %d: %s\n", created.ID, summarizeTest(req))
	return 0
}

func readTestRequest(path string) (CreateTestRequest, error) {
	var r io.Reader = os.Stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return CreateTestRequest{}, err
		}
		defer f.Close()
		r = f
	}
	var req CreateTestRequest
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return CreateTestRequest{}, fmt.Errorf("read test body: %w", err)
	}
	return req, nil
}

// summarizeTest describes a validated request in one line, e.g.
// "reading test1 (cambridge 20 test 1): 38 questions, 40 marks, 38 explained, 38 with evidence".
func summarizeTest(req CreateTestRequest) string {
	s := req.Skill + " " + req.TaskType
	if req.Series != "" {
		s += fmt.Sprintf(" (%s %d test %d)", req.Series, req.Volume, req.TestNumber)
	}
	questions, err := questionsFromContent(req.Skill, req.ContentData)
	if err == nil && len(questions) > 0 {
		marks, explained, cited := 0, 0, 0
		for _, q := range questions {
			marks += q.Span
			if q.Explanation != "" {
				explained++
			}
			if len(q.Evidence) > 0 {
				cited++
			}
		}
		s += fmt.Sprintf(": %d questions, %d marks, %d explained, %d with evidence", len(questions), marks, explained, cited)
	}
	if req.Skill == "writing" {
		var c WritingContent
		if json.Unmarshal(req.ContentData, &c) == nil {
			image := "no image"
			if c.ImageURL != "" {
				image = "image " + c.ImageURL
			}
			s += fmt.Sprintf(": prompt %d words, %s, sample answer %d words", countWords(c.Prompt), image, countWords(c.SampleAnswer))
		}
	}
	return s
}
