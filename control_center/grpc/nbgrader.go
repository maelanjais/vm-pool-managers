package grpc

import (
	"bytes"
	"control_center/config"
	"control_center/internal/sshinject"
	"control_center/models"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// nbgraderSSHClient dials the instructor VM for a given pool and returns a connected SSH client.
// The caller is responsible for closing the client.
func nbgraderSSHClient(poolID, userID string) (*ssh.Client, error) {
	var pool models.Serverpool
	if err := config.Database.
		Where("serverpool_id = ? AND user_id = ?", poolID, userID).
		First(&pool).Error; err != nil {
		return nil, fmt.Errorf("pool not found: %w", err)
	}

	// Find the instructor VM (there should be only one, locked to the instructor).
	var server models.Server
	if err := config.Database.
		Where("serverpool_id = ? AND user_id = ?", poolID, userID).
		Order("created_at ASC").
		First(&server).Error; err != nil {
		return nil, fmt.Errorf("instructor VM not found: %w", err)
	}
	if server.IP_Address == "" {
		return nil, fmt.Errorf("instructor VM has no IP address")
	}

	keyPath := os.Getenv("SSH_PRIVATE_KEY_PATH")
	if keyPath == "" {
		keyPath = os.Getenv("SSH_KEY_PATH")
	}
	signer, err := sshinject.LoadPrivateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("load SSH key: %w", err)
	}

	cfg := sshinject.SshConfig("vmuser", signer)
	client, err := ssh.Dial("tcp", server.IP_Address+":22", cfg)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %s: %w", server.IP_Address, err)
	}
	return client, nil
}

// dockerExec wraps a command to run inside the 'jupyter' Docker container as jovyan.
// Falls back to direct execution if Docker is not available.
func dockerExec(cmd string) string {
	nativeCmd := strings.ReplaceAll(cmd, "/home/jovyan/", "/home/vmuser/")
	nativeCmd = "export PATH=/home/vmuser/jupyter-env/bin:$PATH && " + nativeCmd
	qCmdDocker := shellQuote(cmd)
	qCmdNative := shellQuote(nativeCmd)
	return fmt.Sprintf(`if sudo docker ps | grep -q jupyter 2>/dev/null; then sudo docker exec jupyter bash -c %s 2>&1; else bash -c %s 2>&1; fi`, qCmdDocker, qCmdNative)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// runSSHOutput runs a command via SSH and returns its stdout as a string.
func runSSHOutput(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		return "", fmt.Errorf("run %q: %w (stderr: %s)", cmd, err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// handleNbgraderAssignments lists released assignments on the instructor VM.
// GET /api/nbgrader/assignments?pool_id=X&user_id=Y
func handleNbgraderAssignments(w http.ResponseWriter, r *http.Request) {
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	if poolID == "" || userID == "" {
		http.Error(w, "missing pool_id or user_id", http.StatusBadRequest)
		return
	}

	client, err := nbgraderSSHClient(poolID, userID)
	if err != nil {
		log.Printf("[nbgrader] assignments SSH error: %v", err)
		http.Error(w, "cannot connect to instructor VM: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer client.Close()

	// List assignment directories in nbgrader/source/ inside the jupyter container.
	// Only directories under source/ are assignments — never fall back to the home
	// dir (that listed repo files like Manifest.toml/README.md as fake assignments).
	out, err := runSSHOutput(client, dockerExec(`find /home/jovyan/nbgrader/source/ -mindepth 1 -maxdepth 1 -type d -printf '%f\n' 2>/dev/null || true`))
	if err != nil {
		http.Error(w, "SSH command failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var assignments []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			assignments = append(assignments, line)
		}
	}
	if assignments == nil {
		assignments = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"assignments": assignments})
}

// handleNbgraderCollect runs `nbgrader collect` on the instructor VM.
// POST /api/nbgrader/collect?pool_id=X&user_id=Y&assignment=Z
func handleNbgraderCollect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	assignment := r.URL.Query().Get("assignment")
	if poolID == "" || userID == "" || assignment == "" {
		http.Error(w, "missing pool_id, user_id or assignment", http.StatusBadRequest)
		return
	}

	instrClient, err := nbgraderSSHClient(poolID, userID)
	if err != nil {
		instrClient, err = nbgraderSSHClientAny(poolID, userID)
		if err != nil {
			http.Error(w, "cannot connect to instructor VM: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	defer instrClient.Close()

	var pool models.Serverpool
	if err := config.Database.Where("serverpool_id = ? AND user_id = ?", poolID, userID).First(&pool).Error; err != nil {
		http.Error(w, "pool not found", http.StatusNotFound)
		return
	}

	var list models.ListStudents
	config.Database.Preload("Students").Where("pool_id = ?", pool.ID).First(&list)

	keyPath := os.Getenv("SSH_PRIVATE_KEY_PATH")
	if keyPath == "" { keyPath = os.Getenv("SSH_KEY_PATH") }
	signer, err := sshinject.LoadPrivateKey(keyPath)
	if err != nil {
		http.Error(w, "load SSH key: "+err.Error(), http.StatusInternalServerError)
		return
	}

	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05.000000 MST")
	collected := 0
	var distErrors []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, student := range list.Students {
		if student.IP == "" { continue }
		wg.Add(1)
		go func(s models.Student) {
			defer wg.Done()
			cfg := sshinject.SshConfig("vmuser", signer)
			studentClient, err := ssh.Dial("tcp", s.IP+":22", cfg)
			if err != nil { return }
			defer studentClient.Close()

			// Get files from student's submitted_copies/<assignment> or fallback to nbgrader/<assignment>
			fileListInner := fmt.Sprintf(`find /home/vmuser/nbgrader/submitted_copies/%s -type f 2>/dev/null | sed "s|/home/vmuser/nbgrader/submitted_copies/%s/||" || find /home/vmuser/nbgrader/%s -type f 2>/dev/null | sed "s|/home/vmuser/nbgrader/%s/||" || echo ""`, assignment, assignment, assignment, assignment)
			fileList, err := runSSHOutput(studentClient, fileListInner)
			if err != nil || fileList == "" { return }

			// Create submitted directory on instructor
			mkdirCmd := dockerExec(fmt.Sprintf("mkdir -p /home/jovyan/nbgrader/submitted/%s/%s", s.Name, assignment))
			runSSHOutput(instrClient, mkdirCmd)
			
			// Add student to nbgrader DB if not exists
			addStudentCmd := dockerExec(fmt.Sprintf("cd /home/jovyan/nbgrader && nbgrader db student add %s 2>/dev/null || true", s.Name))
			runSSHOutput(instrClient, addStudentCmd)

			filesFound := 0
			for _, relFile := range strings.Split(strings.TrimSpace(fileList), "\n") {
				relFile = strings.TrimSpace(relFile)
				if relFile == "" { continue }
				
				// Read from student
				catCmd := fmt.Sprintf("cat /home/vmuser/nbgrader/submitted_copies/%s/%s 2>/dev/null || cat /home/vmuser/nbgrader/%s/%s", assignment, relFile, assignment, relFile)
				content, readErr := runSSHOutput(studentClient, catCmd)
				if readErr != nil { continue }
				
				// Ensure dest dir exists
				destPath := fmt.Sprintf("/home/vmuser/nbgrader/submitted/%s/%s/%s", s.Name, assignment, relFile)
				runSSHOutput(instrClient, fmt.Sprintf("mkdir -p $(dirname %q)", destPath))

				// Write to instructor via SCP helper
				scpWriteFile(instrClient, destPath, []byte(content))
				filesFound++
			}

			if filesFound > 0 {
				// Create timestamp.txt
				tsPath := fmt.Sprintf("/home/vmuser/nbgrader/submitted/%s/%s/timestamp.txt", s.Name, assignment)
				scpWriteFile(instrClient, tsPath, []byte(timestamp))
				
				mu.Lock()
				collected++
				mu.Unlock()
			}
		}(student)
	}
	wg.Wait()

	out := fmt.Sprintf("Collected %d submissions.\n", collected)
	if len(distErrors) > 0 {
		out += "Errors:\n" + strings.Join(distErrors, "\n")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"output": out,
	})
}

// handleNbgraderAutograde runs `nbgrader autograde` on the instructor VM.
// POST /api/nbgrader/autograde?pool_id=X&user_id=Y&assignment=Z
func handleNbgraderAutograde(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	assignment := r.URL.Query().Get("assignment")
	if poolID == "" || userID == "" || assignment == "" {
		http.Error(w, "missing pool_id, user_id or assignment", http.StatusBadRequest)
		return
	}

	client, err := nbgraderSSHClient(poolID, userID)
	if err != nil {
		log.Printf("[nbgrader] autograde SSH error: %v", err)
		http.Error(w, "cannot connect to instructor VM: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer client.Close()

	cmd := dockerExec(fmt.Sprintf("cd /home/jovyan/nbgrader && nbgrader autograde %s", assignment))
	out, err := runSSHOutput(client, cmd)
	if err != nil {
		log.Printf("[nbgrader] autograde error: %v", err)
		// Return partial output even on error (nbgrader may exit non-zero but still grade)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "error",
			"output": out + "\n" + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     "ok",
		"assignment": assignment,
		"output":     out,
	})
}

// NbgraderGrade represents one student's grade for an assignment.
type NbgraderGrade struct {
	Student  string  `json:"student"`
	Score    float64 `json:"score"`
	MaxScore float64 `json:"max_score"`
	Status   string  `json:"status"` // "graded", "missing", "needs_manual_grade"
}

// handleNbgraderGrades reads the gradebook via `nbgrader export` CSV on the instructor VM.
// GET /api/nbgrader/grades?pool_id=X&user_id=Y&assignment=Z
func handleNbgraderGrades(w http.ResponseWriter, r *http.Request) {
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	assignment := r.URL.Query().Get("assignment")
	if poolID == "" || userID == "" || assignment == "" {
		http.Error(w, "missing pool_id, user_id or assignment", http.StatusBadRequest)
		return
	}

	grades, err := fetchNbgraderGrades(poolID, userID, assignment)
	if err != nil {
		log.Printf("[nbgrader] grades SSH error: %v", err)
		http.Error(w, "cannot connect to instructor VM: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"grades": grades})
}

// fetchNbgraderGrades lit les notes d'un assignment sur la VM instructeur (nbgrader export
// CSV, fallback sqlite gradebook.db). Réutilisé par le push vers Moodle.
func fetchNbgraderGrades(poolID, userID, assignment string) ([]NbgraderGrade, error) {
	client, err := nbgraderSSHClient(poolID, userID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	tmpFile := fmt.Sprintf("/tmp/nbgrader_export_%d.csv", time.Now().UnixNano())
	innerGrades := fmt.Sprintf("cd /home/jovyan/nbgrader && nbgrader export --to=%s && cat %s; rm -f %s", tmpFile, tmpFile, tmpFile)
	out, err := runSSHOutput(client, dockerExec(innerGrades))
	go func() {
		s, _ := client.NewSession()
		if s != nil {
			s.Run(fmt.Sprintf("rm -f %q", tmpFile))
			s.Close()
		}
	}()

	if err != nil || out == "" {
		// Fallback: gradebook.db via sqlite3
		sqlInner := fmt.Sprintf(
			`sqlite3 /home/jovyan/nbgrader/gradebook.db "SELECT s.name, nb.score, nb.max_score FROM grade nb JOIN student s ON nb.student_id = s.id JOIN notebook n ON nb.notebook_id = n.id JOIN assignment a ON n.assignment_id = a.id WHERE a.name='%s' ORDER BY s.name;" 2>/dev/null || echo ""`,
			strings.ReplaceAll(assignment, "'", "''"),
		)
		out2, err2 := runSSHOutput(client, dockerExec(sqlInner))
		if err2 != nil || out2 == "" {
			return []NbgraderGrade{}, nil
		}
		return parseSQLiteGrades(out2), nil
	}
	return parseCSVGrades(out, assignment), nil
}

// POST /api/nbgrader/submit?pool_id=X&user_id=Y&assignment=Z&student_ip=W
func handleNbgraderSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	assignment := r.URL.Query().Get("assignment") // optional
	studentIP := r.URL.Query().Get("student_ip")
	if poolID == "" || userID == "" || studentIP == "" {
		http.Error(w, "missing pool_id, user_id, or student_ip", http.StatusBadRequest)
		return
	}

	keyPath := os.Getenv("SSH_PRIVATE_KEY_PATH")
	if keyPath == "" { keyPath = os.Getenv("SSH_KEY_PATH") }
	signer, err := sshinject.LoadPrivateKey(keyPath)
	if err != nil {
		http.Error(w, "load SSH key: "+err.Error(), http.StatusInternalServerError)
		return
	}

	cfg := sshinject.SshConfig("vmuser", signer)
	studentClient, err := ssh.Dial("tcp", studentIP+":22", cfg)
	if err != nil {
		http.Error(w, "ssh dial: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer studentClient.Close()

	var cmd string
	// chmod -R a-w (not 444): keep read+EXECUTE so directories stay traversable.
	// 444 strips the x bit on dirs -> "Permission denied" -> collect finds nothing.
	if assignment != "" {
		cmd = fmt.Sprintf("mkdir -p ~/nbgrader/submitted_copies/%q && cp -r ~/nbgrader/%q/* ~/nbgrader/submitted_copies/%q/ 2>/dev/null || true; chmod -R a-w ~/nbgrader/submitted_copies/%q || true", assignment, assignment, assignment, assignment)
	} else {
		cmd = "mkdir -p ~/nbgrader/submitted_copies && rsync -av --exclude='submitted_copies' ~/nbgrader/ ~/nbgrader/submitted_copies/ 2>/dev/null || true; chmod -R a-w ~/nbgrader/submitted_copies || true"
	}
	if err := sshinject.RunSSHcmd(studentClient, cmd); err != nil {
		http.Error(w, "submit copy failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
}

// handleNbgraderSubmissionURL resolves the formgrader URL to manually grade a
// specific student's submission. Formgrader has no /manage_submissions/<a>/<student>
// route — manual grading lives at /formgrader/submissions/<submission_id>/?index=0,
// where submission_id is the gradebook UUID for (assignment, student).
// GET /api/nbgrader/submission-url?pool_id=X&user_id=Y&assignment=Z&student=W
func handleNbgraderSubmissionURL(w http.ResponseWriter, r *http.Request) {
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	assignment := r.URL.Query().Get("assignment")
	student := r.URL.Query().Get("student")
	if poolID == "" || userID == "" || assignment == "" || student == "" {
		http.Error(w, "missing pool_id, user_id, assignment or student", http.StatusBadRequest)
		return
	}

	client, err := nbgraderSSHClient(poolID, userID)
	if err != nil {
		client, err = nbgraderSSHClientAny(poolID, userID)
		if err != nil {
			http.Error(w, "cannot connect to instructor VM: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	defer client.Close()

	// formgrader grades per NOTEBOOK: /formgrader/submissions/<submitted_notebook.id>.
	// (submitted_assignment.id gives a 404 — wrong granularity.)
	q := fmt.Sprintf(
		`sqlite3 /home/jovyan/nbgrader/gradebook.db "SELECT sn.id FROM submitted_notebook sn JOIN submitted_assignment sa ON sn.assignment_id=sa.id JOIN assignment a ON sa.assignment_id=a.id WHERE a.name='%s' AND sa.student_id='%s' LIMIT 1;" 2>/dev/null || echo ""`,
		strings.ReplaceAll(assignment, "'", "''"), strings.ReplaceAll(student, "'", "''"),
	)
	out, _ := runSSHOutput(client, dockerExec(q))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"submission_id": strings.TrimSpace(out)})
}

// parseCSVGrades parses nbgrader CSV export (columns: student_id,assignment,score,max_score,needs_manual_grade)
func parseCSVGrades(csv, assignment string) []NbgraderGrade {
	var grades []NbgraderGrade
	lines := strings.Split(csv, "\n")
	
	headerIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "assignment,") || strings.Contains(line, "student_id") {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 || len(lines) <= headerIdx+1 {
		return grades
	}

	header := strings.Split(lines[headerIdx], ",")
	idx := func(name string) int {
		for i, h := range header {
			if strings.TrimSpace(h) == name {
				return i
			}
		}
		return -1
	}
	iStudent := idx("student_id")
	iAssign := idx("assignment")
	iScore := idx("score")
	iMax := idx("max_score")
	iNMG := idx("needs_manual_grade")
	iTimestamp := idx("timestamp") // vide => l'étudiant n'a rien rendu
	if iStudent < 0 || iScore < 0 {
		return grades
	}

	for _, line := range lines[headerIdx+1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		cols := strings.Split(line, ",")
		if len(cols) <= iStudent {
			continue
		}
		if iAssign >= 0 && len(cols) > iAssign && strings.TrimSpace(cols[iAssign]) != assignment {
			continue
		}
		g := NbgraderGrade{Student: strings.TrimSpace(cols[iStudent])}
		if iScore >= 0 && len(cols) > iScore {
			fmt.Sscanf(cols[iScore], "%f", &g.Score)
		}
		if iMax >= 0 && len(cols) > iMax {
			fmt.Sscanf(cols[iMax], "%f", &g.MaxScore)
		}
		g.Status = "graded"
		// Pas de timestamp de soumission => l'étudiant n'a pas rendu (note non significative).
		if iTimestamp >= 0 && len(cols) > iTimestamp && strings.TrimSpace(cols[iTimestamp]) == "" {
			g.Status = "missing"
		} else if iNMG >= 0 && len(cols) > iNMG && strings.TrimSpace(cols[iNMG]) == "True" {
			g.Status = "needs_manual_grade"
		}
		grades = append(grades, g)
	}
	return grades
}

// handleNbgraderJupyterURL returns the JupyterLab URL for the instructor VM.
// GET /api/nbgrader/jupyter-url?pool_id=X&user_id=Y
func handleNbgraderJupyterURL(w http.ResponseWriter, r *http.Request) {
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	if poolID == "" || userID == "" {
		http.Error(w, "missing pool_id or user_id", http.StatusBadRequest)
		return
	}

	var server models.Server
	if err := config.Database.
		Where("serverpool_id = ? AND user_id = ?", poolID, userID).
		Order("created_at ASC").
		First(&server).Error; err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"url": ""})
		return
	}

	ip := server.IP_Address
	if server.Networks != nil {
		for _, net := range server.Networks {
			if idx := strings.LastIndex(net, ":"); idx >= 0 {
				ip = net[idx+1:]
				break
			}
		}
	}

	// Get app_port from the pool (defaults to 8888 for JupyterLab)
	var pool models.Serverpool
	port := 8888
	if err := config.Database.Where("serverpool_id = ? AND user_id = ?", poolID, userID).First(&pool).Error; err == nil {
		if pool.AppPort > 0 {
			port = pool.AppPort
		}
	}

	// Encode @ explicitly for Caddy proxy URL since Caddy rejects it in path segments
	encodedUserID := strings.ReplaceAll(url.PathEscape(userID), "@", "%40")
	proxyURL := fmt.Sprintf("/api/jupyter-proxy/%s/%s", poolID, encodedUserID)
	
	// Use direct connection without proxy path since base_url is not configured
	directURL := fmt.Sprintf("http://%s:%d", ip, port)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url":       proxyURL + "/",
		"directUrl": directURL,
	})
}

// handleNbgraderRelease releases an assignment from the instructor VM to all student VMs.
// POST /api/nbgrader/release?pool_id=X&user_id=Y&assignment=Z
func handleNbgraderRelease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	assignment := r.URL.Query().Get("assignment")
	if poolID == "" || userID == "" || assignment == "" {
		http.Error(w, "missing pool_id, user_id or assignment", http.StatusBadRequest)
		return
	}

	// 1. SSH on instructor VM → nbgrader release
	instrClient, err := nbgraderSSHClient(poolID, userID)
	if err != nil {
		// Fallback: try any pool with this name (non-instructor pools can also have JupyterLab)
		instrClient, err = nbgraderSSHClientAny(poolID, userID)
		if err != nil {
			log.Printf("[nbgrader] release SSH error: %v", err)
			http.Error(w, "cannot connect to instructor VM: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	defer instrClient.Close()

	// Générer la version distribuable AVANT de distribuer : si l'enseignant a seulement
	// créé/marqué les cellules sans cliquer "Generate", release/<a> serait vide.
	// generate_assignment (peuple release/) puis release_assignment (ancien nom: release).
	releaseInner := fmt.Sprintf("cd /home/jovyan/nbgrader && (nbgrader generate_assignment %s --force 2>&1 || nbgrader assign %s --force 2>&1); nbgrader release_assignment %s 2>&1 || nbgrader release %s 2>&1", assignment, assignment, assignment, assignment)
	releaseOut, err := runSSHOutput(instrClient, dockerExec(releaseInner))
	if err != nil {
		log.Printf("[nbgrader] release command error: %v", err)
		// Don't fail — la version générée peut déjà exister dans release/.
	}

	// 2. Read released files from instructor VM
	fileListInner := fmt.Sprintf(`find /home/jovyan/nbgrader/release/%s -type f 2>/dev/null | sed "s|/home/jovyan/nbgrader/release/%s/||" || echo ""`, assignment, assignment)
	fileList, err := runSSHOutput(instrClient, dockerExec(fileListInner))
	if err != nil || fileList == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"output":  releaseOut,
			"message": "No files found in release directory",
		})
		return
	}

	// 3. Get instructor VM IP for SCP
	var instrServer models.Server
	config.Database.Where("serverpool_id = ? AND user_id = ?", poolID, userID).Order("created_at ASC").First(&instrServer)

	// 4. Get SSH key
	keyPath := os.Getenv("SSH_PRIVATE_KEY_PATH")
	if keyPath == "" {
		keyPath = os.Getenv("SSH_KEY_PATH")
	}
	signer, err := sshinject.LoadPrivateKey(keyPath)
	if err != nil {
		http.Error(w, "load SSH key: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. Find all student VMs for this pool
	var pool models.Serverpool
	if err := config.Database.Where("serverpool_id = ? AND user_id = ?", poolID, userID).First(&pool).Error; err != nil {
		http.Error(w, "pool not found", http.StatusNotFound)
		return
	}

	var list models.ListStudents
	config.Database.Preload("Students").Where("pool_id = ?", pool.ID).First(&list)

	distributed := 0
	var distErrors []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, student := range list.Students {
		if student.IP == "" {
			continue
		}
		wg.Add(1)
		go func(s models.Student) {
			defer wg.Done()
			cfg := sshinject.SshConfig("vmuser", signer)
			studentClient, err := ssh.Dial("tcp", s.IP+":22", cfg)
			if err != nil {
				mu.Lock()
				distErrors = append(distErrors, fmt.Sprintf("%s: SSH dial failed: %v", s.Name, err))
				mu.Unlock()
				return
			}
			defer studentClient.Close()

			// Create assignment directory
			mkdirCmd := fmt.Sprintf("mkdir -p ~/nbgrader/%q", assignment)
			if err := sshinject.RunSSHcmd(studentClient, mkdirCmd); err != nil {
				mu.Lock()
				distErrors = append(distErrors, fmt.Sprintf("%s: mkdir failed: %v", s.Name, err))
				mu.Unlock()
				return
			}

			// SCP files from instructor to student via control center (read from instructor, write to student)
			for _, relFile := range strings.Split(strings.TrimSpace(fileList), "\n") {
				relFile = strings.TrimSpace(relFile)
				if relFile == "" {
					continue
				}
				// Read file from instructor VM
				// Read from Docker container via SSH pipe
				catCmd := dockerExec(fmt.Sprintf("cat /home/jovyan/nbgrader/release/%s/%s", assignment, relFile))
				content, readErr := func() ([]byte, error) {
					out, e := runSSHOutput(instrClient, catCmd)
					return []byte(out), e
				}()
				if readErr != nil {
					mu.Lock()
					distErrors = append(distErrors, fmt.Sprintf("%s: read %s failed: %v", s.Name, relFile, readErr))
					mu.Unlock()
					continue
				}
				// Write file to student VM
				destPath := fmt.Sprintf("/home/vmuser/nbgrader/%s/%s", assignment, relFile)
				if writeErr := scpWriteFile(studentClient, destPath, content); writeErr != nil {
					mu.Lock()
					distErrors = append(distErrors, fmt.Sprintf("%s: write %s failed: %v", s.Name, relFile, writeErr))
					mu.Unlock()
				}
			}

			mu.Lock()
			distributed++
			mu.Unlock()
		}(student)
	}
	// Garde-fou : ne jamais bloquer le bouton indéfiniment, même si une VM ne répond pas.
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(90 * time.Second):
		mu.Lock()
		distErrors = append(distErrors, "délai dépassé : certaines machines n'ont pas répondu à temps")
		mu.Unlock()
	}

	mu.Lock()
	distributedSnapshot := distributed
	errorsSnapshot := append([]string{}, distErrors...)
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":      "ok",
		"distributed": distributedSnapshot,
		"errors":      errorsSnapshot,
		"output":      releaseOut,
	})
}

// scpReadFile reads a remote file via SSH/SCP and returns its content.
func scpReadFile(client *ssh.Client, remotePath string) ([]byte, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	var buf bytes.Buffer
	session.Stdout = &buf
	if err := session.Run(fmt.Sprintf("cat %q", remotePath)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// scpWriteFile writes content to a remote file via SSH.
func scpWriteFile(client *ssh.Client, remotePath string, content []byte) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Ensure parent directory exists
	dir := remotePath[:strings.LastIndex(remotePath, "/")]
	mkSession, _ := client.NewSession()
	if mkSession != nil {
		mkSession.Run(fmt.Sprintf("mkdir -p %q", dir))
		mkSession.Close()
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("cat > %q", remotePath)
	if err := session.Start(cmd); err != nil {
		return err
	}
	if _, err := io.Copy(stdin, bytes.NewReader(content)); err != nil {
		return err
	}
	stdin.Close()
	return session.Wait()
}

// nbgraderSSHClientAny connects to any VM in the pool (not just instructor role).
func nbgraderSSHClientAny(poolID, userID string) (*ssh.Client, error) {
	var server models.Server
	if err := config.Database.
		Where("serverpool_id = ? AND user_id = ?", poolID, userID).
		Order("created_at ASC").
		First(&server).Error; err != nil {
		return nil, fmt.Errorf("no VM found for pool %s/%s: %w", poolID, userID, err)
	}
	if server.IP_Address == "" {
		return nil, fmt.Errorf("VM has no IP address")
	}
	keyPath := os.Getenv("SSH_PRIVATE_KEY_PATH")
	if keyPath == "" {
		keyPath = os.Getenv("SSH_KEY_PATH")
	}
	signer, err := sshinject.LoadPrivateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("load SSH key: %w", err)
	}
	cfg := sshinject.SshConfig("vmuser", signer)
	return ssh.Dial("tcp", server.IP_Address+":22", cfg)
}

// handleNbgraderExportCSV exports grades as a downloadable CSV file.
// GET /api/nbgrader/export-csv?pool_id=X&user_id=Y&assignment=Z
func handleNbgraderExportCSV(w http.ResponseWriter, r *http.Request) {
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	assignment := r.URL.Query().Get("assignment")
	if poolID == "" || userID == "" {
		http.Error(w, "missing pool_id or user_id", http.StatusBadRequest)
		return
	}

	client, err := nbgraderSSHClient(poolID, userID)
	if err != nil {
		client, err = nbgraderSSHClientAny(poolID, userID)
		if err != nil {
			http.Error(w, "cannot connect to instructor VM: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	defer client.Close()

	tmpFile := fmt.Sprintf("/tmp/nbgrader_export_%d.csv", time.Now().UnixNano())
	exportInner := fmt.Sprintf("cd /home/jovyan/nbgrader && nbgrader export --to=%s 2>/dev/null && cat %s; rm -f %s", tmpFile, tmpFile, tmpFile)
	out, err := runSSHOutput(client, dockerExec(exportInner))
	if err != nil || out == "" {
		// Fallback: build CSV from sqlite
		whereClause := ""
		if assignment != "" {
			whereClause = fmt.Sprintf(" WHERE a.name='%s'", strings.ReplaceAll(assignment, "'", "''"))
		}
		sqlCsvInner := fmt.Sprintf(
			`sqlite3 /home/jovyan/nbgrader/gradebook.db "SELECT s.name, a.name, nb.score, nb.max_score FROM grade nb JOIN student s ON nb.student_id = s.id JOIN notebook n ON nb.notebook_id = n.id JOIN assignment a ON n.assignment_id = a.id%s ORDER BY s.name;" 2>/dev/null || echo ""`,
			whereClause,
		)
		sqlOut, _ := runSSHOutput(client, dockerExec(sqlCsvInner))
		if sqlOut == "" {
			http.Error(w, "no grades available", http.StatusNotFound)
			return
		}
		var sb strings.Builder
		sb.WriteString("student_id,assignment,score,max_score\n")
		for _, line := range strings.Split(sqlOut, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				parts := strings.Split(line, "|")
				if len(parts) == 4 {
					sb.WriteString(strings.Join(parts, ",") + "\n")
				}
			}
		}
		out = sb.String()
	}

	filename := "grades"
	if assignment != "" {
		filename = "grades_" + assignment
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, filename))
	w.Write([]byte(out))
}

// parseSQLiteGrades parses sqlite3 pipe-separated output: name|score|max_score
func parseSQLiteGrades(out string) []NbgraderGrade {
	var grades []NbgraderGrade
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		g := NbgraderGrade{Student: strings.TrimSpace(parts[0]), Status: "graded"}
		fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &g.Score)
		fmt.Sscanf(strings.TrimSpace(parts[2]), "%f", &g.MaxScore)
		grades = append(grades, g)
	}
	return grades
}
