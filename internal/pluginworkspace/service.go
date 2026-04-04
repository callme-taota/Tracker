package pluginworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"Tracker/internal/core"
	"Tracker/internal/plugin"
	"Tracker/internal/pluginruntime"
	"Tracker/internal/storage"
)

type Service struct {
	root string
	db   *storage.DB
	eng  *core.Engine
}

type ReviewFinding struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
}

type ReviewReport struct {
	Status    string          `json:"status"`
	RiskLevel string          `json:"risk_level"`
	Summary   string          `json:"summary"`
	Findings  []ReviewFinding `json:"findings"`
}

type ImportRequest struct {
	PluginID     string            `json:"plugin_id"`
	Name         string            `json:"name"`
	Runtime      string            `json:"runtime"`
	Version      string            `json:"version"`
	ManifestJSON string            `json:"manifest_json"`
	EntryFile    string            `json:"entry_file"`
	BuildCommand string            `json:"build_command"`
	RunCommand   string            `json:"run_command"`
	Files        map[string]string `json:"files"`
}

type ExportBundle struct {
	Package storage.PluginPackage        `json:"package"`
	Version storage.PluginPackageVersion `json:"version"`
	Files   map[string]string            `json:"files"`
}

func New(db *storage.DB, eng *core.Engine) *Service {
	root := strings.TrimSpace(os.Getenv("TRACKER_PLUGIN_WORKSPACE_DIR"))
	if root == "" {
		root = ".tracker/plugins"
	}
	return &Service{root: root, db: db, eng: eng}
}

func (s *Service) Root() string { return s.root }

func (s *Service) EnsureWorkspace() error {
	return os.MkdirAll(s.root, 0o755)
}

func sanitizeSegment(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, "..", "")
	v = strings.ReplaceAll(v, "/", "_")
	v = strings.ReplaceAll(v, "\\", "_")
	if v == "" {
		v = "default"
	}
	return v
}

func (s *Service) versionDir(pluginID, version string) string {
	return filepath.Join(s.root, sanitizeSegment(pluginID), sanitizeSegment(version))
}

func safeJoin(base, rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("path required")
	}
	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", fmt.Errorf("invalid path")
	}
	full := filepath.Join(base, clean)
	baseClean := filepath.Clean(base) + string(os.PathSeparator)
	fullClean := filepath.Clean(full)
	if fullClean != filepath.Clean(base) && !strings.HasPrefix(fullClean, baseClean) {
		return "", fmt.Errorf("invalid path")
	}
	return fullClean, nil
}

func (s *Service) WriteFiles(dir string, files map[string]string) error {
	for rel, content := range files {
		full, err := safeJoin(dir, rel)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ReadAllFiles(dir string) (map[string]string, error) {
	files := map[string]string{}
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(b)
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

func (s *Service) ReadFile(pkg storage.PluginPackage, ver storage.PluginPackageVersion, rel string) (string, error) {
	full, err := safeJoin(ver.CodeDir, rel)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Service) WriteFile(pkg storage.PluginPackage, ver storage.PluginPackageVersion, rel, content string) error {
	full, err := safeJoin(ver.CodeDir, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), 0o644)
}

func (s *Service) DeleteFile(pkg storage.PluginPackage, ver storage.PluginPackageVersion, rel string) error {
	full, err := safeJoin(ver.CodeDir, rel)
	if err != nil {
		return err
	}
	return os.Remove(full)
}

func computeChecksum(files map[string]string) string {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(files[k]))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (s *Service) ReviewImport(req ImportRequest) ReviewReport {
	report := ReviewReport{
		Status:    "approved",
		RiskLevel: "low",
		Summary:   "未发现阻断性问题",
	}
	add := func(sev, code, msg, path string) {
		report.Findings = append(report.Findings, ReviewFinding{Severity: sev, Code: code, Message: msg, Path: path})
		switch sev {
		case "high":
			report.Status = "needs_review"
			report.RiskLevel = "high"
			report.Summary = "存在高风险命令或代码模式，需管理员审查"
		case "medium":
			if report.RiskLevel != "high" {
				report.RiskLevel = "medium"
			}
			if report.Status == "approved" {
				report.Status = "needs_review"
				report.Summary = "存在需要人工确认的风险项"
			}
		}
	}
	if req.PluginID == "" || req.Name == "" || req.Runtime == "" {
		add("high", "missing_metadata", "插件基础元数据不完整", "")
	}
	if req.ManifestJSON == "" {
		add("high", "missing_manifest", "缺少 manifest_json", "")
	} else {
		var man plugin.Manifest
		if err := json.Unmarshal([]byte(req.ManifestJSON), &man); err != nil {
			add("high", "invalid_manifest", "manifest_json 不是有效 JSON", "")
		} else {
			if man.ID != "" && man.ID != req.PluginID {
				add("high", "manifest_id_mismatch", "manifest.id 与 plugin_id 不一致", "")
			}
			if man.Kind == plugin.TypeOperator {
				add("medium", "operator_plugin", "operator 类型插件需额外审查权限边界", "")
			}
		}
	}
	for _, v := range []string{req.BuildCommand, req.RunCommand} {
		l := strings.ToLower(v)
		if strings.Contains(l, "curl ") || strings.Contains(l, "wget ") {
			add("high", "network_fetch_command", "构建/运行命令包含动态下载行为", "")
		}
		if strings.Contains(l, "sh -c") || strings.Contains(l, "bash -c") {
			add("high", "shell_exec", "构建/运行命令包含 shell -c，需重点审查", "")
		}
	}
	for path, content := range req.Files {
		l := strings.ToLower(content)
		if strings.Contains(l, "os/exec") || strings.Contains(l, "child_process") || strings.Contains(l, "subprocess") {
			add("high", "subprocess_usage", "源码含子进程执行能力", path)
		}
		if strings.Contains(l, "eval(") || strings.Contains(l, "new function(") {
			add("high", "dynamic_eval", "源码含动态执行逻辑", path)
		}
		if strings.Contains(l, "http://") || strings.Contains(l, "https://") {
			add("medium", "network_usage", "源码包含外部网络访问痕迹", path)
		}
	}
	return report
}

func defaultBuildCommand(runtime string) string {
	switch runtime {
	case "go":
		return "go build -o plugin-bin ."
	case "ts":
		return ""
	case "rust":
		return "cargo build --release"
	default:
		return ""
	}
}

func defaultRunCommand(runtime, entryFile string) string {
	switch runtime {
	case "go":
		return "go run ."
	case "ts":
		if entryFile != "" {
			return "deno run -A " + entryFile
		}
		return "deno run -A main.ts"
	case "rust":
		return "cargo run --release"
	default:
		return ""
	}
}

func splitCommand(cmd string) []string {
	return strings.Fields(strings.TrimSpace(cmd))
}

func (s *Service) Import(ctx context.Context, req ImportRequest) (*storage.PluginPackage, *storage.PluginPackageVersion, ReviewReport, error) {
	if s.db == nil {
		return nil, nil, ReviewReport{}, fmt.Errorf("storage not configured")
	}
	if err := s.EnsureWorkspace(); err != nil {
		return nil, nil, ReviewReport{}, err
	}
	if req.Version == "" {
		req.Version = "0.1.0"
	}
	report := s.ReviewImport(req)
	dir := s.versionDir(req.PluginID, req.Version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, report, err
	}
	if err := s.WriteFiles(dir, req.Files); err != nil {
		return nil, nil, report, err
	}
	reportJSON, _ := json.Marshal(report)
	checksum := computeChecksum(req.Files)
	pkg := storage.PluginPackage{
		PluginID:     req.PluginID,
		Name:         req.Name,
		Runtime:      req.Runtime,
		SourceKind:   "imported",
		ReviewStatus: report.Status,
		RiskLevel:    report.RiskLevel,
		Enabled:      false,
	}
	ver := storage.PluginPackageVersion{
		Version:          req.Version,
		ManifestJSON:     req.ManifestJSON,
		EntryFile:        req.EntryFile,
		BuildCommand:     strings.TrimSpace(req.BuildCommand),
		RunCommand:       strings.TrimSpace(req.RunCommand),
		ReviewReportJSON: string(reportJSON),
		SourceChecksum:   checksum,
		CodeDir:          dir,
	}
	if ver.BuildCommand == "" {
		ver.BuildCommand = defaultBuildCommand(req.Runtime)
	}
	if ver.RunCommand == "" {
		ver.RunCommand = defaultRunCommand(req.Runtime, req.EntryFile)
	}
	pkgID, verID, err := s.db.CreatePluginPackage(pkg, ver)
	if err != nil {
		return nil, nil, report, err
	}
	created, _ := s.db.GetPluginPackage(pkgID)
	createdVer, _ := s.db.GetPluginPackageVersion(verID)
	_ = ctx
	return created, createdVer, report, nil
}

func (s *Service) Build(ctx context.Context, ver storage.PluginPackageVersion) (string, error) {
	cmdParts := splitCommand(ver.BuildCommand)
	if len(cmdParts) == 0 {
		return "未配置 build command，已跳过。", nil
	}
	cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	cmd.Dir = ver.CodeDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (s *Service) SyncRuntime(ctx context.Context) error {
	if s.db == nil || s.eng == nil || s.eng.PluginHost == nil {
		return nil
	}
	pkgs, err := s.db.ListPluginPackages()
	if err != nil {
		return err
	}
	var specs []pluginruntime.PluginSpec
	for _, pkg := range pkgs {
		if !pkg.Enabled || pkg.ReviewStatus != "approved" {
			continue
		}
		ver, err := s.db.GetCurrentPluginPackageVersion(pkg.ID)
		if err != nil || ver == nil {
			continue
		}
		cmd := splitCommand(ver.RunCommand)
		if len(cmd) == 0 {
			cmd = splitCommand(defaultRunCommand(pkg.Runtime, ver.EntryFile))
		}
		if len(cmd) == 0 {
			continue
		}
		specs = append(specs, pluginruntime.PluginSpec{
			ID:      pkg.PluginID,
			Command: cmd,
			Workdir: ver.CodeDir,
			Env: map[string]string{
				"TRACKER_PLUGIN_ENTRY": ver.EntryFile,
			},
		})
	}
	if err := s.eng.PluginHost.SetExtraSpecs(ctx, specs); err != nil {
		return err
	}
	return s.eng.PluginHost.InitRemote(ctx, s.eng.PluginHost.LastGlobalConfig())
}

func (s *Service) BuildAndSync(ctx context.Context, pkg storage.PluginPackage, ver storage.PluginPackageVersion) (string, error) {
	out, err := s.Build(ctx, ver)
	if err != nil {
		return out, err
	}
	return out, s.SyncRuntime(ctx)
}

func (s *Service) Export(pkg storage.PluginPackage, ver storage.PluginPackageVersion) (*ExportBundle, error) {
	files, err := s.ReadAllFiles(ver.CodeDir)
	if err != nil {
		return nil, err
	}
	return &ExportBundle{Package: pkg, Version: ver, Files: files}, nil
}

func (s *Service) TouchRunReview(ctx context.Context, pkg storage.PluginPackage) error {
	pkg.Enabled = true
	if pkg.ReviewStatus == "" {
		pkg.ReviewStatus = "approved"
	}
	if pkg.RiskLevel == "" {
		pkg.RiskLevel = "medium"
	}
	if err := s.db.UpdatePluginPackage(pkg); err != nil {
		return err
	}
	return s.SyncRuntime(ctx)
}
