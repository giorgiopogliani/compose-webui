package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

//go:embed all:dist
var dist embed.FS

type serviceStatus struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`
	Health string `json:"health"`
	Ports  string `json:"ports"`
	Status string `json:"status"`
}

type project struct {
	Name        string          `json:"name"`
	Status      string          `json:"status"`
	ConfigFiles []string        `json:"configFiles"`
	WorkingDir  string          `json:"workingDir"`
	Services    []serviceStatus `json:"services"`
}

type projectsResponse struct {
	Projects []project `json:"projects"`
	Error    string    `json:"error,omitempty"`
}

type actionResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

var (
	nameRe    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	timeout   time.Duration
	psTimeout time.Duration
)

func composeRun(ctx context.Context, dir string, files []string, projectName string, args ...string) (string, error) {
	cmdArgs := []string{"compose"}
	for _, f := range files {
		cmdArgs = append(cmdArgs, "-f", f)
	}
	if projectName != "" {
		cmdArgs = append(cmdArgs, "-p", projectName)
	}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func composeLs(ctx context.Context) ([]project, error) {
	out, err := composeRun(ctx, "", nil, "", "ls", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, out)
	}
	var raw []struct {
		Name        string `json:"Name"`
		Status      string `json:"Status"`
		ConfigFiles string `json:"ConfigFiles"`
		WorkingDir  string `json:"WorkingDir"`
	}
	if out == "" || out == "null" {
		return []project{}, nil
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, fmt.Errorf("parse `compose ls` output: %w: %s", err, out)
	}
	projects := make([]project, 0, len(raw))
	for _, r := range raw {
		files := []string{}
		for _, f := range strings.Split(r.ConfigFiles, ",") {
			if f = strings.TrimSpace(f); f != "" {
				files = append(files, f)
			}
		}
		projects = append(projects, project{
			Name:        r.Name,
			Status:      r.Status,
			ConfigFiles: files,
			WorkingDir:  r.WorkingDir,
			Services:    []serviceStatus{},
		})
	}
	return projects, nil
}

func composePs(ctx context.Context, p *project) error {
	ctx, cancel := context.WithTimeout(ctx, psTimeout)
	defer cancel()

	dir := p.WorkingDir
	if dir == "" && len(p.ConfigFiles) > 0 {
		dir = fileDir(p.ConfigFiles[0])
	}
	out, err := composeRun(ctx, dir, p.ConfigFiles, p.Name, "ps", "--format", "json", "--all")
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	var raw []struct {
		Name    string `json:"Name"`
		Image   string `json:"Image"`
		State   string `json:"State"`
		Health  string `json:"Health"`
		Ports   string `json:"Ports"`
		Status  string `json:"Status"`
		Service string `json:"Service"`
	}
	if out == "" || out == "null" {
		return nil
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		// compose v5 prints newline-delimited JSON objects, older versions print an array.
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var item struct {
				Name    string `json:"Name"`
				Image   string `json:"Image"`
				State   string `json:"State"`
				Health  string `json:"Health"`
				Ports   string `json:"Ports"`
				Status  string `json:"Status"`
				Service string `json:"Service"`
			}
			if err := json.Unmarshal([]byte(line), &item); err != nil {
				return fmt.Errorf("parse `compose ps` output: %w: %s", err, out)
			}
			raw = append(raw, item)
		}
	}
	for _, r := range raw {
		name := r.Service
		if name == "" {
			name = r.Name
		}
		p.Services = append(p.Services, serviceStatus{
			Name:   name,
			Image:  r.Image,
			State:  r.State,
			Health: r.Health,
			Ports:  r.Ports,
			Status: r.Status,
		})
	}
	return nil
}

func fileDir(path string) string {
	if i := strings.LastIndex(path, "/"); i > 0 {
		return path[:i]
	}
	return ""
}

func handleProjects(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), psTimeout)
	defer cancel()

	projects, err := composeLs(ctx)
	if err != nil {
		writeJSON(w, http.StatusOK, projectsResponse{Projects: []project{}, Error: err.Error()})
		return
	}
	projectNames := map[string]bool{}
	for i := range projects {
		if projectNames[projects[i].Name] {
			continue
		}
		projectNames[projects[i].Name] = true
		if err := composePs(ctx, &projects[i]); err != nil {
			projects[i].Status = "error"
		}
	}
	writeJSON(w, http.StatusOK, projectsResponse{Projects: projects})
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	service := r.PathValue("service")
	action := r.PathValue("action")
	if !nameRe.MatchString(name) || (service != "" && !nameRe.MatchString(service)) {
		writeJSON(w, http.StatusBadRequest, actionResponse{OK: false, Message: "invalid name"})
		return
	}
	if action != "up" && action != "restart" {
		writeJSON(w, http.StatusBadRequest, actionResponse{OK: false, Message: "unknown action"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	projects, err := composeLs(ctx)
	if err != nil {
		writeJSON(w, http.StatusOK, actionResponse{OK: false, Message: err.Error()})
		return
	}
	var target *project
	for i := range projects {
		if projects[i].Name == name {
			target = &projects[i]
			break
		}
	}
	if target == nil {
		writeJSON(w, http.StatusNotFound, actionResponse{OK: false, Message: fmt.Sprintf("project %q not found", name)})
		return
	}

	dir := target.WorkingDir
	if dir == "" && len(target.ConfigFiles) > 0 {
		dir = fileDir(target.ConfigFiles[0])
	}
	args := []string{action}
	if action == "up" {
		args = append(args, "-d")
	}
	if service != "" {
		args = append(args, service)
	}
	out, err := composeRun(ctx, dir, target.ConfigFiles, target.Name, args...)
	if err != nil {
		writeJSON(w, http.StatusOK, actionResponse{OK: false, Message: truncate(out, 4000)})
		return
	}
	writeJSON(w, http.StatusOK, actionResponse{OK: true, Message: "ok"})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func main() {
	host := flag.String("host", "127.0.0.1", "address to listen on")
	port := flag.String("port", "8123", "port to listen on")
	flag.DurationVar(&timeout, "timeout", 10*time.Minute, "timeout for up/restart commands")
	flag.DurationVar(&psTimeout, "status-timeout", 30*time.Second, "timeout for status listing commands")
	flag.Parse()

	distFS, err := fs.Sub(dist, "dist")
	if err != nil {
		log.Fatalf("frontend not built: %v (run `pnpm build`)", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/projects", handleProjects)
	mux.HandleFunc("POST /api/projects/{name}/{action}", handleAction)
	mux.HandleFunc("POST /api/projects/{name}/services/{service}/{action}", handleAction)
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	addr := fmt.Sprintf("%s:%s", *host, *port)
	log.Printf("compose-dash listening on http://%s", addr)
	if err := http.ListenAndServe(addr, logging(mux)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
