package grpc

import (
	"strings"
	"testing"
)

func TestShellQuote(t *testing.T) {
	if got := shellQuote("simple"); got != "'simple'" {
		t.Errorf("shellQuote(simple) = %q", got)
	}
	// Une apostrophe doit être échappée façon shell (fermer/échapper/rouvrir).
	if got := shellQuote("a'b"); got != `'a'"'"'b'` {
		t.Errorf("shellQuote(a'b) = %q", got)
	}
}

func TestDockerExec(t *testing.T) {
	out := dockerExec("cd /home/jovyan/nbgrader && ls")
	if !strings.Contains(out, "sudo docker exec jupyter") {
		t.Errorf("dockerExec doit cibler le conteneur jupyter: %q", out)
	}
	// Le fallback natif réécrit le chemin jovyan -> vmuser.
	if !strings.Contains(out, "/home/vmuser/nbgrader") {
		t.Errorf("dockerExec doit fournir un fallback /home/vmuser: %q", out)
	}
}

func TestParseCSVGrades(t *testing.T) {
	csv := strings.Join([]string{
		"assignment,duedate,timestamp,student_id,first_name,last_name,email,max_score,score,needs_manual_grade",
		"TP1,,2026-01-01 10:00,alice@example.com,,,,26,18,False",
		"TP1,,,bob@example.com,,,,26,0,False", // pas de timestamp => non rendu
		"TP1,,2026-01-01 11:00,charlie@example.com,,,,26,5,True", // à corriger
		"TP2,,2026-01-01 10:00,alice@example.com,,,,10,9,False", // autre assignment => ignoré
	}, "\n")

	grades := parseCSVGrades(csv, "TP1")
	if len(grades) != 3 {
		t.Fatalf("attendu 3 notes pour TP1, obtenu %d (%+v)", len(grades), grades)
	}
	by := map[string]NbgraderGrade{}
	for _, g := range grades {
		by[g.Student] = g
	}
	if g := by["alice@example.com"]; g.Score != 18 || g.MaxScore != 26 || g.Status != "graded" {
		t.Errorf("alice = %+v", g)
	}
	if g := by["bob@example.com"]; g.Status != "missing" {
		t.Errorf("bob devrait être 'missing' (pas de timestamp): %+v", g)
	}
	if g := by["charlie@example.com"]; g.Status != "needs_manual_grade" {
		t.Errorf("charlie devrait être 'needs_manual_grade': %+v", g)
	}
}

func TestParseSQLiteGrades(t *testing.T) {
	grades := parseSQLiteGrades("alice@example.com|18.0|26.0\nbob@example.com|0|26\n")
	if len(grades) != 2 {
		t.Fatalf("attendu 2 notes, obtenu %d", len(grades))
	}
	if grades[0].Student != "alice@example.com" || grades[0].Score != 18 || grades[0].MaxScore != 26 {
		t.Errorf("ligne sqlite mal parsée: %+v", grades[0])
	}
}
