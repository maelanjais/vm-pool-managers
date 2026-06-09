package grpc

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"control_center/config"
	"control_center/internal/attribvm"
	"control_center/internal/moodle"
	"control_center/models"
)

// writeJSON est un petit helper de réponse JSON.
func writeJSONMoodle(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// GET /api/moodle/status — indique si Moodle est configuré (pour activer l'UI conditionnellement).
func handleMoodleStatus(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{"configured": moodle.Configured()}
	if c, err := moodle.New(); err == nil {
		resp["url"] = c.BaseHost()
	}
	writeJSONMoodle(w, http.StatusOK, resp)
}

// GET /api/moodle/courses — liste les cours Moodle (pour le sélecteur d'import).
func handleMoodleCourses(w http.ResponseWriter, r *http.Request) {
	c, err := moodle.New()
	if err != nil {
		writeJSONMoodle(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	courses, err := c.GetCourses()
	if err != nil {
		writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{"courses": courses})
}

type moodleStudentDTO struct {
	MoodleID  int    `json:"moodle_id"`
	Email     string `json:"email"`
	FullName  string `json:"fullname"`
	IsTeacher bool   `json:"is_teacher"`
}

// GET /api/moodle/enrolments?course_id=X — élèves inscrits (aperçu avant import).
func handleMoodleEnrolments(w http.ResponseWriter, r *http.Request) {
	courseID, err := strconv.Atoi(r.URL.Query().Get("course_id"))
	if err != nil || courseID <= 0 {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "course_id invalide"})
		return
	}
	c, err := moodle.New()
	if err != nil {
		writeJSONMoodle(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	users, err := c.GetEnrolledUsers(courseID)
	if err != nil {
		writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	out := make([]moodleStudentDTO, 0, len(users))
	for _, u := range users {
		out = append(out, moodleStudentDTO{
			MoodleID: u.ID, Email: u.Email, FullName: u.FullName, IsTeacher: u.IsTeacher(),
		})
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{"students": out})
}

type moodleImportRequest struct {
	PoolID   string   `json:"pool_id"`
	UserID   string   `json:"user_id"`
	CourseID int      `json:"course_id"`
	Emails   []string `json:"emails"` // optionnel : restreint l'import à ces emails
}

// POST /api/moodle/import — importe les élèves d'un cours Moodle dans un pool.
// Crée une ligne students par élève (Name = email = id nbgrader, MoodleEmail, MoodleUserID),
// sans clé SSH (l'accès se fait via JupyterLab/Guacamole ; clé ajoutable plus tard).
func handleMoodleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONMoodle(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST requis"})
		return
	}
	var req moodleImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "JSON invalide"})
		return
	}
	if req.PoolID == "" || req.UserID == "" || req.CourseID <= 0 {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "pool_id, user_id et course_id requis"})
		return
	}

	c, err := moodle.New()
	if err != nil {
		writeJSONMoodle(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	users, err := c.GetEnrolledUsers(req.CourseID)
	if err != nil {
		writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	// Filtre optionnel par emails sélectionnés.
	var only map[string]bool
	if len(req.Emails) > 0 {
		only = map[string]bool{}
		for _, e := range req.Emails {
			only[strings.ToLower(strings.TrimSpace(e))] = true
		}
	}

	// Pool + liste d'étudiants.
	var pool models.Serverpool
	if err := config.Database.Preload("ListStudents.Students").
		Where("serverpool_id = ? AND user_id = ?", req.PoolID, req.UserID).
		First(&pool).Error; err != nil {
		writeJSONMoodle(w, http.StatusNotFound, map[string]string{"error": "pool introuvable"})
		return
	}
	list := &pool.ListStudents
	if list.ID == 0 {
		list.PoolId = pool.ID
		if err := config.Database.Create(list).Error; err != nil {
			writeJSONMoodle(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	existing := map[string]bool{}
	for _, s := range list.Students {
		if s.MoodleEmail != "" {
			existing[strings.ToLower(s.MoodleEmail)] = true
		}
	}

	imported, skipped := 0, 0
	for _, u := range users {
		if u.IsTeacher() || u.Email == "" {
			continue
		}
		key := strings.ToLower(u.Email)
		if only != nil && !only[key] {
			continue
		}
		if existing[key] {
			skipped++
			continue
		}
		student := models.Student{
			ListId:       list.ID,
			Name:         u.Email, // = identifiant nbgrader
			MoodleEmail:  u.Email,
			MoodleUserID: u.ID,
		}
		if err := config.Database.Create(&student).Error; err != nil {
			writeJSONMoodle(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		existing[key] = true
		imported++
	}

	// Mémorise le lien pool ↔ cours Moodle (pour le push de notes).
	config.Database.Model(&models.Serverpool{}).Where("id = ?", pool.ID).
		Update("moodle_course_id", req.CourseID)

	writeJSONMoodle(w, http.StatusOK, map[string]any{
		"imported": imported, "skipped": skipped, "course_id": req.CourseID,
	})
}

// POST /api/moodle/login {username, password} — valide les identifiants Moodle,
// crée une session légère et renvoie {session_id, email, fullname, role}.
func handleMoodleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONMoodle(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST requis"})
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "identifiant et mot de passe requis"})
		return
	}
	c, err := moodle.New()
	if err != nil {
		writeJSONMoodle(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	// 1) Valide les identifiants : un token renvoyé = identifiants corrects.
	if _, err := c.LoginToken(req.Username, req.Password, "moodle_mobile_app"); err != nil {
		writeJSONMoodle(w, http.StatusUnauthorized, map[string]string{"error": "identifiants Moodle invalides"})
		return
	}
	// 2) Récupère l'identité via le token de service (le token utilisateur n'a pas accès à l'email).
	u, err := c.UserByUsername(req.Username)
	if err != nil || u.Email == "" {
		writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": "profil Moodle introuvable"})
		return
	}
	// Login Moodle = flux étudiant (les enseignants/admins utilisent OIDC).
	role := "student"

	sessionID := randomState()
	config.Database.Create(&models.MoodleSession{
		ID: sessionID, Email: u.Email, FullName: u.FullName, MoodleUserID: u.ID, Role: role,
	})
	// Purge des sessions de plus de 24 h.
	config.Database.Where("created_at < ?", time.Now().Add(-24*time.Hour)).Delete(&models.MoodleSession{})

	writeJSONMoodle(w, http.StatusOK, map[string]any{
		"session_id": sessionID, "email": u.Email, "fullname": u.FullName, "role": role,
	})
}

// GET /api/moodle/session?id= — renvoie l'identité d'une session Moodle.
func handleMoodleSession(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "id manquant"})
		return
	}
	var sess models.MoodleSession
	if err := config.Database.First(&sess, "id = ?", id).Error; err != nil {
		writeJSONMoodle(w, http.StatusNotFound, map[string]string{"error": "session introuvable"})
		return
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{
		"email": sess.Email, "fullname": sess.FullName, "role": sess.Role,
	})
}

// GET /api/moodle/my-pools?email= — pools où cet email (Moodle) est inscrit.
func handleMoodleMyPools(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "email manquant"})
		return
	}
	type poolRow struct {
		PoolID string `json:"pool_id"`
		UserID string `json:"user_id"`
	}
	var pools []poolRow
	config.Database.Raw(`
		SELECT DISTINCT sp.serverpool_id AS pool_id, sp.user_id AS user_id
		FROM serverpools sp
		JOIN list_students ls ON ls.pool_id = sp.id
		JOIN students st ON st.list_id = ls.id
		WHERE LOWER(st.moodle_email) = LOWER(?) AND sp.serverpool_id <> ''`, email).Scan(&pools)
	writeJSONMoodle(w, http.StatusOK, map[string]any{"pools": pools})
}

// POST /api/moodle/ssh-key {email, ssh_key} — ajoute/maj la clé SSH d'un élève (login Moodle).
func handleMoodleSSHKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONMoodle(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST requis"})
		return
	}
	var req struct {
		Email  string `json:"email"`
		SSHKey string `json:"ssh_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.SSHKey == "" {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "email et ssh_key requis"})
		return
	}
	if err := attribvm.New(config.Database).SetStudentKeyByEmail(req.Email, strings.TrimSpace(req.SSHKey)); err != nil {
		writeJSONMoodle(w, http.StatusOK, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{"success": true})
}

// POST /api/moodle/link-pool {pool_id, user_id, course_id} — lie un pool à un cours Moodle
// (sans importer d'élèves), pour permettre le push de notes.
func handleMoodleLinkPool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONMoodle(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST requis"})
		return
	}
	var req struct {
		PoolID   string `json:"pool_id"`
		UserID   string `json:"user_id"`
		CourseID int    `json:"course_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PoolID == "" || req.UserID == "" || req.CourseID <= 0 {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "pool_id, user_id, course_id requis"})
		return
	}
	res := config.Database.Model(&models.Serverpool{}).
		Where("serverpool_id = ? AND user_id = ?", req.PoolID, req.UserID).
		Update("moodle_course_id", req.CourseID)
	if res.Error != nil || res.RowsAffected == 0 {
		writeJSONMoodle(w, http.StatusNotFound, map[string]string{"error": "pool introuvable"})
		return
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{"success": true, "course_id": req.CourseID})
}

// GET /api/moodle/assignments?course_id=X  (ou ?pool_id=&user_id=) — devoirs Moodle.
// Avec pool_id+user_id, le cours est résolu via le pool (champ moodle_course_id).
func handleMoodleAssignments(w http.ResponseWriter, r *http.Request) {
	courseID, _ := strconv.Atoi(r.URL.Query().Get("course_id"))
	if courseID <= 0 {
		poolID, userID := r.URL.Query().Get("pool_id"), r.URL.Query().Get("user_id")
		if poolID != "" && userID != "" {
			var pool models.Serverpool
			if err := config.Database.Where("serverpool_id = ? AND user_id = ?", poolID, userID).First(&pool).Error; err == nil {
				courseID = pool.MoodleCourseID
			}
		}
	}
	if courseID <= 0 {
		// Pas de cours Moodle lié à ce pool : pas une erreur, juste rien à proposer.
		writeJSONMoodle(w, http.StatusOK, map[string]any{"assignments": []any{}, "course_id": 0})
		return
	}
	c, err := moodle.New()
	if err != nil {
		writeJSONMoodle(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	assigns, err := c.GetAssignments(courseID)
	if err != nil {
		writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{"assignments": assigns, "course_id": courseID})
}

// POST /api/moodle/push-grades — remonte les notes nbgrader d'un assignment vers un devoir Moodle.
func handleMoodlePushGrades(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONMoodle(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST requis"})
		return
	}
	var req struct {
		PoolID         string `json:"pool_id"`
		UserID         string `json:"user_id"`
		Assignment     string `json:"assignment"`
		MoodleAssignID int    `json:"moodle_assign_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil ||
		req.PoolID == "" || req.UserID == "" || req.Assignment == "" || req.MoodleAssignID <= 0 {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "pool_id, user_id, assignment et moodle_assign_id requis"})
		return
	}
	c, err := moodle.New()
	if err != nil {
		writeJSONMoodle(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}

	// 1) Notes nbgrader (student = email).
	grades, err := fetchNbgraderGrades(req.PoolID, req.UserID, req.Assignment)
	if err != nil {
		writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": "lecture des notes impossible: " + err.Error()})
		return
	}

	// 2) Map email → moodle_user_id (depuis les élèves importés du pool).
	var pool models.Serverpool
	if err := config.Database.Preload("ListStudents.Students").
		Where("serverpool_id = ? AND user_id = ?", req.PoolID, req.UserID).First(&pool).Error; err != nil {
		writeJSONMoodle(w, http.StatusNotFound, map[string]string{"error": "pool introuvable"})
		return
	}
	uidByEmail := map[string]int{}
	for _, s := range pool.ListStudents.Students {
		if s.MoodleEmail != "" && s.MoodleUserID != 0 {
			uidByEmail[strings.ToLower(s.MoodleEmail)] = s.MoodleUserID
		}
	}

	// 3) Barème du devoir Moodle (pour mettre la note à l'échelle).
	maxGrade := 100.0
	if assigns, e := c.GetAssignments(pool.MoodleCourseID); e == nil {
		for _, a := range assigns {
			if a.ID == req.MoodleAssignID {
				maxGrade = a.MaxGrade
			}
		}
	}

	pushed, skipped := 0, 0
	var failures []string
	for _, g := range grades {
		// Ne pas pousser de note aux étudiants qui n'ont rien rendu.
		if g.Status == "missing" {
			skipped++
			continue
		}
		uid := uidByEmail[strings.ToLower(g.Student)]
		if uid == 0 {
			skipped++
			continue
		}
		grade := g.Score
		if g.MaxScore > 0 {
			grade = g.Score / g.MaxScore * maxGrade
		}
		if err := c.SaveAssignGrade(req.MoodleAssignID, uid, grade, ""); err != nil {
			failures = append(failures, g.Student)
			continue
		}
		pushed++
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{
		"pushed": pushed, "skipped": skipped, "failures": failures, "total": len(grades),
	})
}

// POST /api/moodle/attrib-vm {pool_id, user_id, email} — attribue une VM sans clé SSH.
func handleMoodleAttribVM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONMoodle(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST requis"})
		return
	}
	var req struct {
		PoolID string `json:"pool_id"`
		UserID string `json:"user_id"`
		Email  string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "JSON invalide"})
		return
	}
	svc := attribvm.New(config.Database)
	ip, appPort, err := svc.AttribVMByEmail(req.PoolID, req.UserID, req.Email)
	if err != nil {
		writeJSONMoodle(w, http.StatusOK, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{
		"success": true, "ip": ip, "app_port": appPort,
	})
}
