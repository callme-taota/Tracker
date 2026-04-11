# 发布说明

## 流水线分层

仓库中的 GitHub Actions 按职责拆成三层：

- `CI`：PR 和 `master` 提交时执行 Go 测试、前端构建、Docker smoke build。
- `Release`：推送 `vX.Y.Z` tag 时生成跨平台压缩包、`checksums.txt` 和 GitHub Release。
- `Docker`：推送 `master` 或 `vX.Y.Z` tag 时构建并发布 GHCR 镜像。

## Release 产物契约

每个正式 tag 会生成按平台区分的压缩包，目录结构统一为：

```text
tracker_<version>_<os>_<arch>/
  tracker[.exe]
  web/dist/
  configs/tracker.example.yaml
  configs/pipeline.example.yaml
  .env.example
  web/.env.example
  README.md
  README.zh.md
```

这样解压后即可直接执行 `tracker serve`，不会只看到未构建前端的占位页。

## 如何发版

1. 确保 `master` 上的 CI 为绿色。
2. 创建并推送语义化版本 tag，例如：

```bash
git tag v0.1.0
git push origin v0.1.0
```

3. 等待 `Release` workflow 完成后，在 GitHub Release 页面下载对应平台压缩包。

## Docker 标签策略

- `master` 分支推送：发布 `ghcr.io/<owner>/tracker:main` 和 `ghcr.io/<owner>/tracker:sha-<shortsha>`
- tag `vX.Y.Z`：发布 `ghcr.io/<owner>/tracker:vX.Y.Z` 和 `ghcr.io/<owner>/tracker:latest`

## 安全约束

- Release 和公开 Docker 构建不会注入真实业务密钥。
- `TRACKER_API_KEY`、`OPENAI_API_KEY` 等运行时密钥应在部署阶段配置，而不是打进构建产物。
- 当前 `VITE_TRACKER_API_KEY` 仍支持作为可选构建参数，但默认流水线不会提供其真实值。
