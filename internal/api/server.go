package api

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"Tracker/internal/core"
	"Tracker/internal/model"
	"Tracker/internal/operator"
	"Tracker/internal/plugin"
	"Tracker/internal/pipeline"
	"Tracker/internal/storage"
	"Tracker/plugins/llm_operator"
)

// Server holds engine, storage and config for the HTTP API.
type Server struct {
	Engine     *core.Engine
	DB         *storage.DB
	ConfigPath string
	useFS      fs.FS // optional: serve SPA from this FS (e.g. web/dist)
}

// NewServer creates an API server. DB may be nil. staticFS is optional (e.g. os.DirFS("web/dist")).
func NewServer(eng *core.Engine, db *storage.DB, configPath string, staticFS fs.FS) *Server {
	if eng != nil {
		if b := operator.NewServerBridge(eng, db); b != nil {
			llm_operator.SetBridge(b)
		} else {
			llm_operator.SetBridge(nil)
		}
	}
	return &Server{Engine: eng, DB: db, ConfigPath: configPath, useFS: staticFS}
}

// Handler returns the root http.Handler. API and assets go to mux; other GET requests serve index (SPA fallback).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/sources", s.listSources)
	mux.HandleFunc("POST /api/sources", s.addSource)
	mux.HandleFunc("DELETE /api/sources/{id}", s.deleteSource)
	mux.HandleFunc("GET /api/interests", s.listInterests)
	mux.HandleFunc("POST /api/interests", s.addInterest)
	mux.HandleFunc("DELETE /api/interests/{id}", s.deleteInterest)
	mux.HandleFunc("GET /api/core/ping", s.corePing)
	mux.HandleFunc("GET /api/plugins/{id}/manifest", s.getPluginManifest)
	mux.HandleFunc("POST /api/plugins/{id}/test", s.testPluginConfig)
	mux.HandleFunc("GET /api/plugins", s.listPlugins)
	mux.HandleFunc("POST /api/plugins/external/reload", s.reloadExternalPlugins)
	mux.HandleFunc("POST /api/operator/chat", s.operatorChat)
	mux.HandleFunc("GET /api/items", s.listItems)
	mux.HandleFunc("GET /api/summaries", s.listSummaries)
	mux.HandleFunc("GET /api/stats", s.stats)
	mux.HandleFunc("GET /api/pipeline/status", s.pipelineStatus)
	mux.HandleFunc("POST /api/pipeline/run", s.pipelineRun)
	mux.HandleFunc("GET /api/pipelines", s.listPipelines)
	mux.HandleFunc("POST /api/pipelines", s.createPipeline)
	mux.HandleFunc("GET /api/pipelines/{id}", s.getPipeline)
	mux.HandleFunc("PUT /api/pipelines/{id}", s.updatePipeline)
	mux.HandleFunc("DELETE /api/pipelines/{id}", s.deletePipeline)
	mux.HandleFunc("POST /api/pipelines/{id}/run", s.runPipelineByID)
	mux.HandleFunc("POST /api/pipelines/{id}/run-async", s.enqueuePipelineRun)
	mux.HandleFunc("POST /api/pipelines/{id}/probe", s.probePipeline)
	mux.HandleFunc("POST /api/pipelines/{id}/rerun", s.rerunPipeline)
	mux.HandleFunc("GET /api/jobs/{id}", s.getJob)
	mux.HandleFunc("GET /assets/{path...}", s.serveStatic)
	return &spaHandler{mux: mux, serveIndex: s.serveIndex}
}

// spaHandler routes /api and /assets to mux, everything else (GET) to index for SPA client-side routing.
type spaHandler struct {
	mux        *http.ServeMux
	serveIndex func(http.ResponseWriter, *http.Request)
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/assets") {
		h.mux.ServeHTTP(w, r)
		return
	}
	if r.Method == http.MethodGet {
		h.serveIndex(w, r)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) listSources(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		jsonResponse(w, []storage.Source{})
		return
	}
	list, err := s.DB.ListSources()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []storage.Source{}
	}
	jsonResponse(w, list)
}

func (s *Server) addSource(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		URL    string `json:"url"`
		Type   string `json:"type"`
		Config string `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.URL == "" {
		http.Error(w, "url required", http.StatusBadRequest)
		return
	}
	if body.Type == "" {
		body.Type = "rss"
	}
	id, err := s.DB.AddSource(body.URL, body.Type, body.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]int64{"id": id})
}

func (s *Server) deleteSource(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.DB.DeleteSource(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listInterests(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		jsonResponse(w, []storage.Interest{})
		return
	}
	list, err := s.DB.ListInterests()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []storage.Interest{}
	}
	jsonResponse(w, list)
}

func (s *Server) addInterest(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Name    string `json:"name"`
		Keywords string `json:"keywords"`
		Config  string `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	id, err := s.DB.AddInterest(body.Name, body.Keywords, body.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]int64{"id": id})
}

func (s *Server) deleteInterest(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.DB.DeleteInterest(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) corePing(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	profiles := []string{}
	sp := make(map[string]string)
	if s.Engine != nil && s.Engine.Core != nil && s.Engine.Core.LLM != nil {
		profiles = s.Engine.Core.LLM.ProfileNames()
	}
	if s.Engine != nil && s.Engine.Core != nil && s.Engine.Core.Storage != nil {
		for k, err := range s.Engine.Core.Storage.Ping(ctx) {
			if err != nil {
				sp[k] = err.Error()
			} else {
				sp[k] = "ok"
			}
		}
	}
	jsonResponse(w, map[string]interface{}{
		"llm_profiles": profiles,
		"storage_ping": sp,
	})
}

func (s *Server) getPluginManifest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing plugin id", http.StatusBadRequest)
		return
	}
	m, ok := s.Engine.Hub.Manifest(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}

type pluginTestBody struct {
	Config map[string]interface{} `json:"config"`
}

// testPluginConfig POST /api/plugins/{id}/test — runs plugin.ConfigTester when implemented.
func (s *Server) testPluginConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing plugin id", http.StatusBadRequest)
		return
	}
	p := s.Engine.PM.GetByName(id)
	if p == nil {
		http.NotFound(w, r)
		return
	}
	var body pluginTestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	cfg := plugin.Config(body.Config)
	if cfg == nil {
		cfg = plugin.Config{}
	}
	tester, ok := p.(plugin.ConfigTester)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":        false,
			"supported": false,
			"error":     "plugin does not implement connection test",
		})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := tester.TestConfig(ctx, cfg); err != nil {
		jsonResponse(w, map[string]interface{}{"ok": false, "supported": true, "error": err.Error()})
		return
	}
	jsonResponse(w, map[string]interface{}{"ok": true, "supported": true})
}

func (s *Server) reloadExternalPlugins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	key := os.Getenv("TRACKER_PLUGIN_ADMIN_KEY")
	if key == "" {
		http.Error(w, "TRACKER_PLUGIN_ADMIN_KEY not set", http.StatusServiceUnavailable)
		return
	}
	if strings.TrimSpace(r.Header.Get("X-Tracker-Plugin-Admin-Key")) != key {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.Engine == nil || s.Engine.PluginHost == nil {
		http.Error(w, "plugin host unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := s.Engine.ReloadExternalPlugins(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]interface{}{"ok": true})
}

type operatorChatBody struct {
	Messages      []plugin.ChatMessage `json:"messages"`
	LLMProfile    string               `json:"llm_profile,omitempty"`
	MaxToolRounds int                  `json:"max_tool_rounds,omitempty"`
	Config        map[string]any       `json:"config,omitempty"`
}

func (s *Server) operatorChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	want := os.Getenv("TRACKER_OPERATOR_API_KEY")
	if want == "" {
		http.Error(w, "TRACKER_OPERATOR_API_KEY not set", http.StatusServiceUnavailable)
		return
	}
	if strings.TrimSpace(r.Header.Get("X-Tracker-Operator-Key")) != want {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.Engine == nil || s.DB == nil {
		http.Error(w, "storage required for operator", http.StatusServiceUnavailable)
		return
	}
	if !s.initEngineFromEnv(w) {
		return
	}
	var body operatorChatBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if len(body.Messages) == 0 {
		http.Error(w, "messages required", http.StatusBadRequest)
		return
	}
	p := s.Engine.PM.GetByName("llm_operator")
	if p == nil {
		http.Error(w, "llm_operator not registered", http.StatusInternalServerError)
		return
	}
	op, ok := p.(plugin.OperatorCapability)
	if !ok {
		http.Error(w, "llm_operator missing capability", http.StatusInternalServerError)
		return
	}
	cfg := plugin.Config(body.Config)
	if cfg == nil {
		cfg = plugin.Config{}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	resp, err := op.Chat(ctx, cfg, plugin.OperatorChatRequest{
		Messages:      body.Messages,
		LLMProfile:    body.LLMProfile,
		MaxToolRounds: body.MaxToolRounds,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, resp)
}

func (s *Server) listPlugins(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil {
		jsonResponse(w, []any{})
		return
	}
	all := s.Engine.PM.ListAll()
	type row struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Type    string `json:"type"`
		Runtime string `json:"runtime"`
		Healthy *bool  `json:"healthy,omitempty"`
	}
	list := make([]row, 0, len(all))
	for _, p := range all {
		rt := "builtin"
		if s.Engine.PluginHost != nil && s.Engine.PluginHost.IsRemote(p.Name()) {
			rt = "remote"
		}
		entry := row{Name: p.Name(), Version: p.Version(), Type: string(p.Type()), Runtime: rt}
		if rt == "remote" && s.Engine.PluginHost != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			_, _, err := s.Engine.PluginHost.Health(ctx, p.Name())
			cancel()
			ok := err == nil
			entry.Healthy = &ok
		}
		list = append(list, entry)
	}
	jsonResponse(w, list)
}

func (s *Server) resolvePipelineConfigPath() string {
	configPath := s.ConfigPath
	if configPath == "" {
		configPath = "config.yaml"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			configPath = "configs/default.yaml"
		}
	}
	return configPath
}

func (s *Server) initEngineFromEnv(w http.ResponseWriter) bool {
	global := plugin.Config{
		"api_key":   os.Getenv("OPENAI_API_KEY"),
		"bot_token": os.Getenv("TELEGRAM_BOT_TOKEN"),
		"chat_id":   os.Getenv("TELEGRAM_CHAT_ID"),
	}
	if err := s.Engine.Init(global); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

func (s *Server) persistPipelineItems(items []*model.Item) {
	if s.DB == nil || len(items) == 0 {
		return
	}
	for _, it := range items {
		var sid *int64
		ts := it.Timestamp
		if ts.IsZero() {
			ts = time.Now()
		}
		itemID, _ := s.DB.SaveItem(sid, it.Title, it.URL, it.Content, it.Summary, ts, "")
		if itemID > 0 && (it.Summary != "" || len(it.KeyPoints) > 0) {
			kpJSON := storage.KeyPointsToJSON(it.KeyPoints)
			_, _ = s.DB.SaveSummary(itemID, it.Summary, kpJSON)
		}
	}
}

// resolveDefaultPipelineForStatus returns JSON-friendly status payload (DB default graph or YAML stages).
func (s *Server) resolveDefaultPipelineForStatus() map[string]interface{} {
	if s.DB != nil {
		def, err := s.DB.GetDefaultPipelineDefinition()
		if err == nil && def != nil {
			var graphObj interface{}
			_ = json.Unmarshal([]byte(def.GraphJSON), &graphObj)
			return map[string]interface{}{
				"source":     "db",
				"id":         def.ID,
				"name":       def.Name,
				"is_default": def.IsDefault,
				"status":     "idle",
				"graph":      graphObj,
			}
		}
	}
	configPath := s.resolvePipelineConfigPath()
	pipe, err := pipeline.LoadFromFile(configPath)
	if err != nil {
		return map[string]interface{}{"source": "file", "status": "error", "error": err.Error()}
	}
	stages := make([]map[string]string, 0, len(pipe.Stages))
	for _, st := range pipe.Stages {
		stages = append(stages, map[string]string{
			"name":       st.Name,
			"plugin_id":  st.PluginID,
			"type":       string(st.PluginType),
		})
	}
	return map[string]interface{}{
		"source":       "file",
		"config_path":  configPath,
		"name":         pipe.Name,
		"status":       "idle",
		"stages":       stages,
		"linear_stages": stages,
	}
}

func (s *Server) pipelineStatus(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, s.resolveDefaultPipelineForStatus())
}

func (s *Server) pipelineRun(w http.ResponseWriter, r *http.Request) {
	if !s.initEngineFromEnv(w) {
		return
	}
	if s.DB != nil {
		def, err := s.DB.GetDefaultPipelineDefinition()
		if err == nil && def != nil {
			g, perr := pipeline.ParseGraphJSON([]byte(def.GraphJSON))
			if perr != nil {
				http.Error(w, perr.Error(), http.StatusBadRequest)
				return
			}
			items, err := s.Engine.RunGraph(g)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			s.persistPipelineItems(items)
			jsonResponse(w, map[string]interface{}{"items_processed": len(items), "saved": s.DB != nil, "source": "db"})
			return
		}
	}
	configPath := s.resolvePipelineConfigPath()
	pipe, err := pipeline.LoadFromFile(configPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	items, err := s.Engine.Run(pipe)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.persistPipelineItems(items)
	jsonResponse(w, map[string]interface{}{"items_processed": len(items), "saved": s.DB != nil, "source": "file"})
}

func (s *Server) listPipelines(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		jsonResponse(w, []storage.PipelineSummary{})
		return
	}
	list, err := s.DB.ListPipelineDefinitions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []storage.PipelineSummary{}
	}
	jsonResponse(w, list)
}

func (s *Server) getPipeline(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	row, err := s.DB.GetPipelineDefinition(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if row == nil {
		http.NotFound(w, r)
		return
	}
	var graphObj interface{}
	_ = json.Unmarshal([]byte(row.GraphJSON), &graphObj)
	jsonResponse(w, map[string]interface{}{
		"id": row.ID, "name": row.Name, "is_default": row.IsDefault,
		"created_at": row.CreatedAt.Format(time.RFC3339),
		"updated_at": row.UpdatedAt.Format(time.RFC3339),
		"graph":      graphObj,
	})
}

type pipelineWriteBody struct {
	Name      string          `json:"name"`
	IsDefault bool            `json:"is_default"`
	Graph     json.RawMessage `json:"graph"`
}

func (s *Server) createPipeline(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	var body pipelineWriteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" || len(body.Graph) == 0 {
		http.Error(w, "name and graph required", http.StatusBadRequest)
		return
	}
	g, err := pipeline.ParseGraphJSON(body.Graph)
	if err != nil {
		http.Error(w, "invalid graph: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Engine.Hub.ValidatePipelineGraph(g); err != nil {
		http.Error(w, "pluginhub: "+err.Error(), http.StatusBadRequest)
		return
	}
	graphJSON, err := g.ToJSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, err := s.DB.CreatePipelineDefinition(body.Name, string(graphJSON), body.IsDefault)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]int64{"id": id})
}

func (s *Server) updatePipeline(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body pipelineWriteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" || len(body.Graph) == 0 {
		http.Error(w, "name and graph required", http.StatusBadRequest)
		return
	}
	g, err := pipeline.ParseGraphJSON(body.Graph)
	if err != nil {
		http.Error(w, "invalid graph: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Engine.Hub.ValidatePipelineGraph(g); err != nil {
		http.Error(w, "pluginhub: "+err.Error(), http.StatusBadRequest)
		return
	}
	graphJSON, err := g.ToJSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.DB.UpdatePipelineDefinition(id, body.Name, string(graphJSON), body.IsDefault); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deletePipeline(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.DB.DeletePipelineDefinition(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) runPipelineByID(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	if !s.initEngineFromEnv(w) {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	row, err := s.DB.GetPipelineDefinition(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if row == nil {
		http.NotFound(w, r)
		return
	}
	g, err := pipeline.ParseGraphJSON([]byte(row.GraphJSON))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	items, err := s.Engine.RunGraphWithContext(r.Context(), id, g)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.persistPipelineItems(items)
	jsonResponse(w, map[string]interface{}{
		"items_processed": len(items), "saved": s.DB != nil, "pipeline_id": id,
	})
}

func (s *Server) enqueuePipelineRun(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	row, err := s.DB.GetPipelineDefinition(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if row == nil {
		http.NotFound(w, r)
		return
	}
	jid, err := s.DB.EnqueueJob(id, "run", nil, 3)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]int64{"job_id": jid})
}

func (s *Server) probePipeline(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	row, err := s.DB.GetPipelineDefinition(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if row == nil {
		http.NotFound(w, r)
		return
	}
	g, err := pipeline.ParseGraphJSON([]byte(row.GraphJSON))
	if err != nil {
		jsonResponse(w, map[string]interface{}{"ok": false, "phase": "parse", "error": err.Error()})
		return
	}
	if err := s.Engine.Hub.ValidatePipelineGraph(g); err != nil {
		jsonResponse(w, map[string]interface{}{"ok": false, "phase": "pluginhub", "error": err.Error()})
		return
	}
	nodes := g.NodeByID()
	order, _ := g.TopologicalOrder()
	if len(order) == 0 {
		for _, n := range g.Nodes {
			order = append(order, n.ID)
		}
	}
	var steps []map[string]interface{}
	for _, nid := range order {
		n := nodes[nid]
		m, _ := s.Engine.Hub.Manifest(n.PluginID)
		entry := map[string]interface{}{
			"node_id": n.ID, "plugin_id": n.PluginID, "plugin_type": string(n.Type),
			"manifest_version": m.Version,
		}
		if err := s.Engine.Hub.ValidateConfig(m, n.Config); err != nil {
			entry["config_ok"] = false
			entry["config_error"] = err.Error()
		} else {
			entry["config_ok"] = true
		}
		if s.Engine.PluginHost != nil && s.Engine.PluginHost.IsRemote(n.PluginID) {
			hctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			status, details, herr := s.Engine.PluginHost.Health(hctx, n.PluginID)
			cancel()
			if herr != nil {
				entry["plugin_health_ok"] = false
				entry["plugin_health_error"] = herr.Error()
			} else {
				entry["plugin_health_ok"] = true
				entry["plugin_health_status"] = status
				if details != "" {
					entry["plugin_health_details"] = details
				}
			}
		}
		steps = append(steps, entry)
	}
	jsonResponse(w, map[string]interface{}{"ok": true, "pipeline_id": id, "nodes": steps})
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	jid, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if jid <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	j, err := s.DB.GetJob(jid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if j == nil {
		http.NotFound(w, r)
		return
	}
	jsonResponse(w, j)
}

// rerunPipeline enqueues a job with time_from / time_to merged into source configs (RFC3339), for backfills.
func (s *Server) rerunPipeline(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	row, err := s.DB.GetPipelineDefinition(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if row == nil {
		http.NotFound(w, r)
		return
	}
	_ = row
	payload := make(map[string]interface{})
	if v := r.URL.Query().Get("from"); v != "" {
		payload["time_from"] = v
	}
	if v := r.URL.Query().Get("to"); v != "" {
		payload["time_to"] = v
	}
	jid, err := s.DB.EnqueueJob(id, "rerun", payload, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]interface{}{"job_id": jid, "pipeline_id": id, "payload": payload})
}

type itemResp struct {
	ID        int64   `json:"id"`
	SourceID  *int64  `json:"source_id"`
	Title     string  `json:"title"`
	URL       string  `json:"url"`
	Content   string  `json:"content"`
	Summary   string  `json:"summary"`
	Timestamp string  `json:"timestamp"`
	Raw       string  `json:"raw"`
	CreatedAt string  `json:"created_at"`
}

func (s *Server) listItems(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		jsonResponse(w, map[string]interface{}{"items": []itemResp{}, "total": 0})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 50
	}
	list, err := s.DB.ListItems(limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	total, _ := s.DB.CountItems()
	if list == nil {
		list = []storage.Item{}
	}
	resp := make([]itemResp, 0, len(list))
	for _, i := range list {
		var sid *int64
		if i.SourceID.Valid {
			sid = &i.SourceID.Int64
		}
		resp = append(resp, itemResp{
			ID: i.ID, SourceID: sid, Title: i.Title, URL: i.URL, Content: i.Content,
			Summary: i.Summary, Timestamp: i.Timestamp.Format(time.RFC3339), Raw: i.Raw,
			CreatedAt: i.CreatedAt.Format(time.RFC3339),
		})
	}
	jsonResponse(w, map[string]interface{}{"items": resp, "total": total})
}

func (s *Server) listSummaries(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		jsonResponse(w, map[string]interface{}{"summaries": []storage.SummaryRow{}, "total": 0})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 50
	}
	list, err := s.DB.ListSummaries(limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	total, _ := s.DB.CountSummaries()
	if list == nil {
		list = []storage.SummaryRow{}
	}
	jsonResponse(w, map[string]interface{}{"summaries": list, "total": total})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	sources, items, interests, summaries := int64(0), int64(0), int64(0), int64(0)
	if s.DB != nil {
		sl, _ := s.DB.ListSources()
		sources = int64(len(sl))
		items, _ = s.DB.CountItems()
		il, _ := s.DB.ListInterests()
		interests = int64(len(il))
		summaries, _ = s.DB.CountSummaries()
	}
	jsonResponse(w, map[string]interface{}{
		"sources":           sources,
		"items":             items,
		"interests":         interests,
		"summaries":         summaries,
		"storage_enabled":   s.DB != nil,
	})
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	if s.useFS != nil {
		data, err := fs.ReadFile(s.useFS, "index.html")
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(placeholderHTML))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	if s.useFS == nil {
		http.NotFound(w, r)
		return
	}
	path := r.PathValue("path")
	if path == "" || strings.Contains(path, "..") || filepath.Clean(path) != path {
		http.NotFound(w, r)
		return
	}
	fullPath := "assets/" + path
	data, err := fs.ReadFile(s.useFS, fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if len(path) > 4 && path[len(path)-4:] == ".css" {
		w.Header().Set("Content-Type", "text/css")
	} else if len(path) > 3 && path[len(path)-3:] == ".js" {
		w.Header().Set("Content-Type", "application/javascript")
	}
	w.Write(data)
}

func jsonResponse(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

const placeholderHTML = `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Tracker</title></head><body><h1>Tracker</h1><p>请先构建前端: <code>cd web && npm install && npm run build</code></p><p>然后重启 <code>./tracker serve</code>。</p></body></html>`
