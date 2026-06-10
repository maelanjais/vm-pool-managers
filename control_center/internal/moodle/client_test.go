package moodle

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newMock construit un client pointant vers un serveur Moodle simulé.
func newMock(handler http.HandlerFunc) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	return &Client{BaseURL: srv.URL, Token: "tok", http: srv.Client()}, srv
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	switch r.FormValue("wsfunction") {
	case "core_course_get_courses_by_field":
		_, _ = w.Write([]byte(`{"courses":[{"id":1,"shortname":"site","fullname":"Site"},{"id":2,"shortname":"PY101","fullname":"Python 101"}]}`))
	case "core_enrol_get_enrolled_users":
		_, _ = w.Write([]byte(`[{"id":3,"fullname":"Alice","email":"alice@x","roles":[{"shortname":"student"}]},{"id":4,"fullname":"Bob","email":"bob@x","roles":[{"shortname":"editingteacher"}]}]`))
	default:
		// Enveloppe d'erreur Moodle (renvoyée en HTTP 200).
		_, _ = w.Write([]byte(`{"exception":"webservice_access_exception","errorcode":"accessexception","message":"Access control exception"}`))
	}
}

func TestGetCoursesFiltersSiteCourse(t *testing.T) {
	c, srv := newMock(wsHandler)
	defer srv.Close()
	courses, err := c.GetCourses()
	if err != nil {
		t.Fatalf("GetCourses: %v", err)
	}
	if len(courses) != 1 || courses[0].ShortName != "PY101" {
		t.Fatalf("le cours-site (id=1) doit être filtré, obtenu %+v", courses)
	}
}

func TestGetEnrolledUsersAndTeacherDetection(t *testing.T) {
	c, srv := newMock(wsHandler)
	defer srv.Close()
	users, err := c.GetEnrolledUsers(2)
	if err != nil {
		t.Fatalf("GetEnrolledUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("attendu 2 inscrits, obtenu %d", len(users))
	}
	if users[0].IsTeacher() {
		t.Errorf("Alice (student) ne doit pas être enseignante")
	}
	if !users[1].IsTeacher() {
		t.Errorf("Bob (editingteacher) doit être détecté enseignant")
	}
}

func TestErrorEnvelopeBecomesError(t *testing.T) {
	c, srv := newMock(wsHandler)
	defer srv.Close()
	// mod_assign_save_grade tombe dans le default => enveloppe d'erreur.
	err := c.SaveAssignGrade(1, 3, 18, "")
	if err == nil {
		t.Fatal("une enveloppe d'erreur Moodle doit produire une erreur Go")
	}
	if !strings.Contains(err.Error(), "Access control") {
		t.Errorf("le message d'erreur Moodle doit être propagé: %v", err)
	}
}

func TestLoginTokenSuccessAndFailure(t *testing.T) {
	c, srv := newMock(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.FormValue("password") == "good" {
			_, _ = w.Write([]byte(`{"token":"abc123"}`))
		} else {
			_, _ = w.Write([]byte(`{"error":"Invalid login","errorcode":"invalidlogin"}`))
		}
	})
	defer srv.Close()

	tok, err := c.LoginToken("alice", "good", "moodle_mobile_app")
	if err != nil || tok != "abc123" {
		t.Fatalf("login valide attendu, got token=%q err=%v", tok, err)
	}
	if _, err := c.LoginToken("alice", "bad", "moodle_mobile_app"); err == nil {
		t.Error("login invalide doit renvoyer une erreur")
	}
}

func TestConfiguredAndNew(t *testing.T) {
	t.Setenv("MOODLE_URL", "")
	t.Setenv("MOODLE_TOKEN", "")
	if Configured() {
		t.Error("non configuré si variables vides")
	}
	if _, err := New(); err == nil {
		t.Error("New doit échouer sans config")
	}
	t.Setenv("MOODLE_URL", "http://localhost:8081/")
	t.Setenv("MOODLE_TOKEN", "tok")
	if !Configured() {
		t.Error("doit être configuré")
	}
	c, err := New()
	if err != nil || c.BaseURL != "http://localhost:8081" { // slash final retiré
		t.Errorf("New: baseURL=%q err=%v", c.BaseURL, err)
	}
}
