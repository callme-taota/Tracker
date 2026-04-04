package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"Tracker/internal/pipeline"
	"Tracker/internal/plugin"
	"Tracker/internal/pluginworkspace"
	"Tracker/internal/storage"
)

func (s *Server) pluginWorkspace() *pluginworkspace.Service {
	return pluginworkspace.New(s.DB, s.Engine)
}

type pluginPackageWriteBody struct {
	PluginID     string            `json:"plugin_id"`
	Name         string            `json:"name"`
	Runtime      string            `json:"runtime"`
	SourceKind   string            `json:"source_kind"`
	Version      string            `json:"version"`
	ManifestJSON string            `json:"manifest_json"`
	EntryFile    string            `json:"entry_file"`
	BuildCommand string            `json:"build_command"`
	RunCommand   string            `json:"run_command"`
	Files        map[string]string `json:"files"`
	Enabled      bool              `json:"enabled"`
}

type pluginFileWriteBody struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type pluginGroupWriteBody struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Graph       json.RawMessage `json:"graph"`
	IO          json.RawMessage `json:"io"`
	ChangeNote  string          `json:"change_note"`
}

type pluginGroupExtractBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PipelineID  int64  `json:"pipeline_id"`
}

func (s *Server) packageResponse(pkg *storage.PluginPackage, ver *storage.PluginPackageVersion) map[string]interface{} {
	runtime := map[string]interface{}{
		"enabled": pkg != nil && pkg.Enabled,
		"loaded":  false,
	}
	if pkg != nil && s != nil && s.Engine != nil && s.Engine.PluginHost != nil {
		loaded := s.Engine.PluginHost.IsRemote(pkg.PluginID)
		runtime["loaded"] = loaded
		if loaded {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			status, details, err := s.Engine.PluginHost.Health(ctx, pkg.PluginID)
			if err != nil {
				runtime["health_error"] = err.Error()
			} else {
				runtime["health_status"] = status
				runtime["health_details"] = details
			}
		}
	}
	return map[string]interface{}{
		"package":         pkg,
		"current_version": ver,
		"runtime_status":  runtime,
	}
}

func pluginRunResponse(pkg *storage.PluginPackage, ok bool, logText string, runErr error) map[string]interface{} {
	out := map[string]interface{}{
		"ok":  ok,
		"log": logText,
	}
	if runErr != nil {
		out["error"] = runErr.Error()
	}
	if pkg != nil {
		out["last_build_status"] = pkg.LastBuildStatus
		out["last_run_status"] = pkg.LastRunStatus
	}
	return out
}

func pluginGroupResponse(g storage.PluginGroup) map[string]interface{} {
	return map[string]interface{}{
		"id":                 g.ID,
		"name":               g.Name,
		"description":        g.Description,
		"graph":              json.RawMessage(g.GraphJSON),
		"io":                 json.RawMessage(g.IOJSON),
		"current_version_id": g.CurrentVersionID,
		"current_version":    g.CurrentVersion,
		"version_count":      g.VersionCount,
		"reference_count":    g.ReferenceCount,
		"outdated_ref_count": g.OutdatedRefCount,
		"created_at":         g.CreatedAt,
		"updated_at":         g.UpdatedAt,
	}
}

func (s *Server) upsertPluginGroup(name, description string, graph, io json.RawMessage) (*storage.PluginGroup, bool, error) {
	group, created, err := s.DB.Services().PluginPlatform.UpsertGroup(name, description, string(graph), string(io))
	if err != nil || group == nil {
		return group, created, err
	}
	refs, outdated, refsErr := s.DB.CountPipelinesReferencingPluginGroup(group.ID)
	if refsErr == nil {
		group.ReferenceCount = refs
		group.OutdatedRefCount = outdated
	}
	return group, created, nil
}

func packageIDFromReq(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func (s *Server) listPluginPackages(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		jsonResponse(w, []storage.PluginPackage{})
		return
	}
	list, err := s.DB.ListPluginPackages()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, list)
}

func (s *Server) createPluginPackage(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	var body pluginPackageWriteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.SourceKind == "" {
		body.SourceKind = "workspace"
	}
	if body.Version == "" {
		body.Version = "0.1.0"
	}
	if body.EntryFile == "" {
		switch body.Runtime {
		case "go":
			body.EntryFile = "main.go"
		case "rust":
			body.EntryFile = "src/main.rs"
		default:
			body.EntryFile = "main.ts"
		}
	}
	if len(body.Files) == 0 {
		body.Files = map[string]string{
			body.EntryFile: pluginTemplate(body.PluginID, body.Runtime),
		}
	}
	ws := s.pluginWorkspace()
	pkg, ver, report, err := ws.Import(r.Context(), pluginworkspace.ImportRequest{
		PluginID:     body.PluginID,
		Name:         body.Name,
		Runtime:      body.Runtime,
		Version:      body.Version,
		ManifestJSON: body.ManifestJSON,
		EntryFile:    body.EntryFile,
		BuildCommand: body.BuildCommand,
		RunCommand:   body.RunCommand,
		Files:        body.Files,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if pkg != nil {
		pkg.SourceKind = body.SourceKind
		pkg.Enabled = body.Enabled
		if report.Status == "approved" && body.Enabled {
			_ = ws.TouchRunReview(r.Context(), *pkg)
			pkg, _ = s.DB.GetPluginPackage(pkg.ID)
		} else {
			_ = s.DB.UpdatePluginPackage(*pkg)
		}
	}
	jsonResponse(w, map[string]interface{}{
		"package":         pkg,
		"current_version": ver,
		"review":          report,
	})
}

func (s *Server) importPluginPackage(w http.ResponseWriter, r *http.Request) {
	s.createPluginPackage(w, r)
}

func (s *Server) getPluginPackage(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	pkg, err := s.DB.GetPluginPackage(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if pkg == nil {
		http.NotFound(w, r)
		return
	}
	ver, err := s.DB.GetCurrentPluginPackageVersion(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, s.packageResponse(pkg, ver))
}

func (s *Server) updatePluginPackage(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	pkg, err := s.DB.GetPluginPackage(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if pkg == nil {
		http.NotFound(w, r)
		return
	}
	ver, err := s.DB.GetCurrentPluginPackageVersion(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if ver == nil {
		http.Error(w, "current version missing", http.StatusConflict)
		return
	}
	var body pluginPackageWriteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.PluginID != "" {
		pkg.PluginID = body.PluginID
	}
	if body.Name != "" {
		pkg.Name = body.Name
	}
	if body.Runtime != "" {
		pkg.Runtime = body.Runtime
	}
	if body.SourceKind != "" {
		pkg.SourceKind = body.SourceKind
	}
	pkg.Enabled = body.Enabled
	if err := s.DB.UpdatePluginPackage(*pkg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if body.ManifestJSON != "" {
		ver.ManifestJSON = body.ManifestJSON
	}
	if body.EntryFile != "" {
		ver.EntryFile = body.EntryFile
	}
	if body.BuildCommand != "" {
		ver.BuildCommand = body.BuildCommand
	}
	if body.RunCommand != "" {
		ver.RunCommand = body.RunCommand
	}
	if err := s.DB.UpdatePluginPackageVersion(*ver); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(body.Files) > 0 {
		if err := s.pluginWorkspace().WriteFiles(ver.CodeDir, body.Files); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if pkg.Enabled {
		_ = s.pluginWorkspace().SyncRuntime(r.Context())
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getPluginPackageFiles(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	ver, err := s.DB.GetCurrentPluginPackageVersion(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if ver == nil {
		http.NotFound(w, r)
		return
	}
	ws := s.pluginWorkspace()
	if path := r.URL.Query().Get("path"); path != "" {
		pkg, _ := s.DB.GetPluginPackage(id)
		content, err := ws.ReadFile(*pkg, *ver, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		jsonResponse(w, map[string]string{"path": path, "content": content})
		return
	}
	files, err := ws.ReadAllFiles(ver.CodeDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, files)
}

func (s *Server) putPluginPackageFile(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	pkg, _ := s.DB.GetPluginPackage(id)
	ver, _ := s.DB.GetCurrentPluginPackageVersion(id)
	if pkg == nil || ver == nil {
		http.NotFound(w, r)
		return
	}
	var body pluginFileWriteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := s.pluginWorkspace().WriteFile(*pkg, *ver, body.Path, body.Content); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deletePluginPackageFile(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	pkg, _ := s.DB.GetPluginPackage(id)
	ver, _ := s.DB.GetCurrentPluginPackageVersion(id)
	if pkg == nil || ver == nil {
		http.NotFound(w, r)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}
	if err := s.pluginWorkspace().DeleteFile(*pkg, *ver, path); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) reviewPluginPackage(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	pkg, _ := s.DB.GetPluginPackage(id)
	ver, _ := s.DB.GetCurrentPluginPackageVersion(id)
	if pkg == nil || ver == nil {
		http.NotFound(w, r)
		return
	}
	files, err := s.pluginWorkspace().ReadAllFiles(ver.CodeDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	report := s.pluginWorkspace().ReviewImport(pluginworkspace.ImportRequest{
		PluginID:     pkg.PluginID,
		Name:         pkg.Name,
		Runtime:      pkg.Runtime,
		Version:      ver.Version,
		ManifestJSON: ver.ManifestJSON,
		EntryFile:    ver.EntryFile,
		BuildCommand: ver.BuildCommand,
		RunCommand:   ver.RunCommand,
		Files:        files,
	})
	reportJSON, _ := json.Marshal(report)
	ver.ReviewReportJSON = string(reportJSON)
	_ = s.DB.UpdatePluginPackageVersion(*ver)
	pkg.ReviewStatus = report.Status
	pkg.RiskLevel = report.RiskLevel
	_ = s.DB.UpdatePluginPackage(*pkg)
	jsonResponse(w, report)
}

func (s *Server) buildPluginPackage(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	pkg, _ := s.DB.GetPluginPackage(id)
	ver, _ := s.DB.GetCurrentPluginPackageVersion(id)
	if pkg == nil || ver == nil {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	logText, err := s.pluginWorkspace().Build(ctx, *ver)
	now := time.Now()
	pkg.LastBuildLog = logText
	pkg.LastBuildAt = &now
	if err != nil {
		pkg.LastBuildStatus = "failed"
		_ = s.DB.UpdatePluginPackage(*pkg)
		jsonResponse(w, pluginRunResponse(pkg, false, logText, err))
		return
	}
	pkg.LastBuildStatus = "success"
	_ = s.DB.UpdatePluginPackage(*pkg)
	jsonResponse(w, pluginRunResponse(pkg, true, logText, nil))
}

func (s *Server) runPluginPackage(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	pkg, _ := s.DB.GetPluginPackage(id)
	ver, _ := s.DB.GetCurrentPluginPackageVersion(id)
	if pkg == nil || ver == nil {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	logText, err := s.pluginWorkspace().BuildAndSync(ctx, *pkg, *ver)
	now := time.Now()
	pkg.LastRunLog = logText
	pkg.LastRunAt = &now
	pkg.LastBuildLog = logText
	pkg.LastBuildAt = &now
	if err != nil {
		pkg.LastBuildStatus = "failed"
		pkg.LastRunStatus = "failed"
		_ = s.DB.UpdatePluginPackage(*pkg)
		jsonResponse(w, pluginRunResponse(pkg, false, logText, err))
		return
	}
	pkg.Enabled = true
	pkg.LastBuildStatus = "success"
	pkg.LastRunStatus = "running"
	if pkg.ReviewStatus == "pending" {
		pkg.ReviewStatus = "approved"
	}
	if pkg.RiskLevel == "unknown" {
		pkg.RiskLevel = "medium"
	}
	_ = s.DB.UpdatePluginPackage(*pkg)
	jsonResponse(w, pluginRunResponse(pkg, true, logText, nil))
}

func (s *Server) stopPluginPackage(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	pkg, _ := s.DB.GetPluginPackage(id)
	if pkg == nil {
		http.NotFound(w, r)
		return
	}
	pkg.Enabled = false
	now := time.Now()
	pkg.LastRunStatus = "stopped"
	pkg.LastRunAt = &now
	if err := s.DB.UpdatePluginPackage(*pkg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.pluginWorkspace().SyncRuntime(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) exportPluginPackage(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	pkg, _ := s.DB.GetPluginPackage(id)
	ver, _ := s.DB.GetCurrentPluginPackageVersion(id)
	if pkg == nil || ver == nil {
		http.NotFound(w, r)
		return
	}
	bundle, err := s.pluginWorkspace().Export(*pkg, *ver)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, bundle)
}

func (s *Server) listPluginGroups(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		jsonResponse(w, []storage.PluginGroup{})
		return
	}
	list, err := s.DB.ListPluginGroups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]map[string]interface{}, 0, len(list))
	for _, g := range list {
		refs, outdated, err := s.DB.CountPipelinesReferencingPluginGroup(g.ID)
		if err == nil {
			g.ReferenceCount = refs
			g.OutdatedRefCount = outdated
		}
		out = append(out, pluginGroupResponse(g))
	}
	jsonResponse(w, out)
}

func (s *Server) createPluginGroup(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	var body pluginGroupWriteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" || len(body.Graph) == 0 {
		http.Error(w, "name and graph required", http.StatusBadRequest)
		return
	}
	group, created, err := s.upsertPluginGroup(body.Name, body.Description, body.Graph, body.IO)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if group == nil {
		http.Error(w, "group not found after save", http.StatusInternalServerError)
		return
	}
	w.Header().Set("X-Tracker-Plugin-Group-Action", map[bool]string{true: "created", false: "updated"}[created])
	jsonResponse(w, pluginGroupResponse(*group))
}

func (s *Server) getPluginGroup(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	row, err := s.DB.GetPluginGroup(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if row == nil {
		http.NotFound(w, r)
		return
	}
	refs, outdated, err := s.DB.CountPipelinesReferencingPluginGroup(row.ID)
	if err == nil {
		row.ReferenceCount = refs
		row.OutdatedRefCount = outdated
	}
	jsonResponse(w, pluginGroupResponse(*row))
}

func (s *Server) updatePluginGroup(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body pluginGroupWriteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" || len(body.Graph) == 0 {
		http.Error(w, "name and graph required", http.StatusBadRequest)
		return
	}
	if err := s.DB.UpdatePluginGroup(id, body.Name, body.Description, string(body.Graph), string(body.IO)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listPluginGroupVersions(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		jsonResponse(w, []storage.PluginGroupVersion{})
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	list, err := s.DB.ListPluginGroupVersions(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]map[string]interface{}, 0, len(list))
	for _, row := range list {
		out = append(out, map[string]interface{}{
			"id":                 row.ID,
			"group_id":           row.GroupID,
			"version":            row.Version,
			"graph":              json.RawMessage(row.GraphJSON),
			"io":                 json.RawMessage(row.IOJSON),
			"source_pipeline_id": row.SourcePipelineID,
			"change_note":        row.ChangeNote,
			"created_at":         row.CreatedAt,
		})
	}
	jsonResponse(w, out)
}

func (s *Server) extractPluginGroupFromPipeline(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	var body pluginGroupExtractBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.PipelineID <= 0 || body.Name == "" {
		http.Error(w, "pipeline_id and name required", http.StatusBadRequest)
		return
	}
	def, err := s.DB.GetPipelineDefinition(body.PipelineID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if def == nil {
		http.NotFound(w, r)
		return
	}
	graph, err := pipeline.ParseGraphJSON([]byte(def.GraphJSON))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rawGraph, _ := graph.ToJSON()
	group, created, err := s.upsertPluginGroup(body.Name, body.Description, rawGraph, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if group == nil {
		http.Error(w, "group not found after save", http.StatusInternalServerError)
		return
	}
	w.Header().Set("X-Tracker-Plugin-Group-Action", map[bool]string{true: "created", false: "updated"}[created])
	jsonResponse(w, pluginGroupResponse(*group))
}

func (s *Server) deletePluginGroup(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := packageIDFromReq(r)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.DB.DeletePluginGroup(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func pluginNeedsConfigTester(p plugin.Plugin, m plugin.Manifest) bool {
	switch m.Kind {
	case plugin.TypeSource, plugin.TypeSummary, plugin.TypeDispatch:
		return true
	case plugin.TypeProcessor:
		return p.Name() == "embedding_dedup" || p.Name() == "llm_event_dedup"
	default:
		return false
	}
}

func kindNeedsExplicitPipelineIO(kind plugin.Type) bool {
	switch kind {
	case plugin.TypeSource, plugin.TypeProcessor, plugin.TypeSummary, plugin.TypeInterest, plugin.TypeDispatch, plugin.TypeOperator:
		return true
	default:
		return false
	}
}

func (s *Server) pluginQualityReport(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil {
		jsonResponse(w, []any{})
		return
	}
	type issue struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	type row struct {
		Name          string  `json:"name"`
		Type          string  `json:"type"`
		Runtime       string  `json:"runtime"`
		Score         int     `json:"score"`
		Category      string  `json:"category"`
		Issues        []issue `json:"issues"`
		HasSchema     bool    `json:"has_schema"`
		HasTester     bool    `json:"has_tester"`
		NeedsTester   bool    `json:"needs_tester"`
		HasPipelineIO bool    `json:"has_pipeline_io"`
	}
	all := s.Engine.PM.ListAll()
	out := make([]row, 0, len(all))
	for _, p := range all {
		m, _ := s.Engine.Hub.Manifest(p.Name())
		score := 100
		issues := make([]issue, 0)
		hasTester := false
		if _, ok := p.(plugin.ConfigTester); ok {
			hasTester = true
		}
		needsTester := pluginNeedsConfigTester(p, m)
		if len(m.ConfigSchema) == 0 {
			score -= 20
			issues = append(issues, issue{Code: "missing_config_schema", Message: "未声明 config_schema"})
		}
		if kindNeedsExplicitPipelineIO(m.Kind) && m.PipelineIO == nil {
			score -= 5
			issues = append(issues, issue{Code: "implicit_pipeline_io", Message: "未显式声明 pipeline_io，仍在依赖推断"})
		}
		if p.Type() != plugin.TypeDispatch && p.Type() != plugin.TypeOperator && len(m.OutputFormats) == 0 {
			score -= 15
			issues = append(issues, issue{Code: "missing_output_formats", Message: "未声明 output_formats"})
		}
		if p.Type() != plugin.TypeSource && len(m.InputFormats) == 0 {
			score -= 10
			issues = append(issues, issue{Code: "missing_input_formats", Message: "未声明 input_formats"})
		}
		if needsTester && !hasTester {
			score -= 15
			issues = append(issues, issue{Code: "missing_config_tester", Message: "该插件依赖外部系统，但未实现 ConfigTester"})
		}
		category := "governance-ready"
		if score < 70 {
			category = "needs-hardening"
		} else if score < 90 {
			category = "needs-improvement"
		}
		runtime := "builtin"
		if s.Engine.PluginHost != nil && s.Engine.PluginHost.IsRemote(p.Name()) {
			runtime = "remote"
		}
		out = append(out, row{
			Name:          p.Name(),
			Type:          string(p.Type()),
			Runtime:       runtime,
			Score:         score,
			Category:      category,
			Issues:        issues,
			HasSchema:     len(m.ConfigSchema) > 0,
			HasTester:     hasTester,
			NeedsTester:   needsTester,
			HasPipelineIO: m.PipelineIO != nil,
		})
	}
	jsonResponse(w, out)
}

func pluginTemplate(pluginID, runtime string) string {
	switch runtime {
	case "go":
		return `package main

import (
	"encoding/json"
	"log"
	"strings"

	srv "Tracker/sdk/plugin-go/server"
)

func main() {
	manifest := json.RawMessage(` + "`" + `{
		"id": "` + pluginID + `",
		"version": "0.1.0",
		"kind": "processor",
		"display_name": "` + pluginID + `",
		"input_formats": ["tracker.item.v1"],
		"output_formats": ["tracker.item.v1"]
	}` + "`" + `)
	h := func(op string, payload json.RawMessage) (interface{}, error) {
		switch op {
		case "handshake":
			var m interface{}
			_ = json.Unmarshal(manifest, &m)
			return map[string]interface{}{"plugin_id": "` + pluginID + `", "version": "0.1.0", "manifest": m}, nil
		case "health", "ready":
			return map[string]string{"status": "ok"}, nil
		case "init", "test_config":
			return map[string]interface{}{}, nil
		case "execute_process":
			var in struct {
				Item map[string]interface{} ` + "`json:\"item\"`" + `
			}
			if err := json.Unmarshal(payload, &in); err != nil {
				return nil, err
			}
			if t, ok := in.Item["title"].(string); ok {
				in.Item["title"] = strings.ToUpper(t)
			}
			return map[string]interface{}{"item": in.Item}, nil
		default:
			return map[string]interface{}{}, nil
		}
	}
	if err := srv.Run(h); err != nil {
		log.Fatal(err)
	}
}
`
	case "rust":
		return `fn main() {
    println!("{\"listen_addr\":\"127.0.0.1:0\"}");
    loop {}
}
`
	default:
		return `const maxFrame = 32 << 20

async function readFull(conn: Deno.Conn, buf: Uint8Array) {
  let o = 0
  while (o < buf.length) {
    const n = await conn.read(buf.subarray(o))
    if (n === null) throw new Error('unexpected EOF')
    o += n
  }
}

async function readFrame(conn: Deno.Conn): Promise<Uint8Array | null> {
  const h = new Uint8Array(4)
  try {
    await readFull(conn, h)
  } catch {
    return null
  }
  const len = (h[0] << 24) | (h[1] << 16) | (h[2] << 8) | h[3]
  const body = new Uint8Array(len)
  await readFull(conn, body)
  return body
}

async function writeFrame(conn: Deno.Conn, obj: unknown) {
  const b = new TextEncoder().encode(JSON.stringify(obj))
  const lb = new Uint8Array(4)
  lb[0] = (b.length >>> 24) & 0xff
  lb[1] = (b.length >>> 16) & 0xff
  lb[2] = (b.length >>> 8) & 0xff
  lb[3] = b.length & 0xff
  await conn.write(lb)
  await conn.write(b)
}

const manifest = {
  id: "` + pluginID + `",
  version: "0.1.0",
  kind: "processor",
  input_formats: ["tracker.item.v1"],
  output_formats: ["tracker.item.v1"],
}

const listener = Deno.listen({ hostname: "127.0.0.1", port: 0 })
const addr = listener.addr as Deno.NetAddr
console.log(JSON.stringify({ listen_addr: addr.hostname + ":" + addr.port }))
const conn = await listener.accept()
while (true) {
  const raw = await readFrame(conn)
  if (raw === null) break
  const req = JSON.parse(new TextDecoder().decode(raw)) as { op: string; payload?: Record<string, unknown> }
  if (req.op === "handshake") {
    await writeFrame(conn, { ok: true, payload: { plugin_id: manifest.id, version: manifest.version, manifest } })
    continue
  }
  if (req.op === "health" || req.op === "ready") {
    await writeFrame(conn, { ok: true, payload: { status: "ok" } })
    continue
  }
  if (req.op === "execute_process") {
    const item = (req.payload?.item ?? {}) as Record<string, unknown>
    await writeFrame(conn, { ok: true, payload: { item } })
    continue
  }
  await writeFrame(conn, { ok: true, payload: {} })
}`
	}
}
