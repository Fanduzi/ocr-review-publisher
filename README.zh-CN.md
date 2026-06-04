# OCR Review Publisher

[![CI](https://github.com/Fanduzi/ocr-review-publisher/actions/workflows/ci.yml/badge.svg)](https://github.com/Fanduzi/ocr-review-publisher/actions/workflows/ci.yml)
![Go Version](https://img.shields.io/badge/go-1.26.1-00ADD8?logo=go)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

[![English](https://img.shields.io/badge/docs-English-blue)](README.md) [![简体中文](https://img.shields.io/badge/docs-简体中文-yellow)](README.zh-CN.md)

[![Quality Gates](https://img.shields.io/badge/Quality_Gates-informational)](docs/quality-gates.zh-CN.md) [![OCR Compatibility](https://img.shields.io/badge/OCR_Compatibility-informational)](docs/ocr-compatibility.zh-CN.md) [![Release Process](https://img.shields.io/badge/Release_Process-success)](docs/release.zh-CN.md) [![GitLab E2E](https://img.shields.io/badge/GitLab_E2E-informational)](docs/e2e-gitlab.zh-CN.md) [![Contributing](https://img.shields.io/badge/Contributing-important)](CONTRIBUTING.md)

`ocr-review-publisher` 是 [Open Code Review](https://github.com/alibaba/open-code-review) 的发布层。Open Code Review 生成代码审查发现，本项目读取其输出并发布为 GitLab MR 评论。

## 背景

我们最初尝试把发布能力直接贡献给 Open Code Review（[PR #15](https://github.com/alibaba/open-code-review/pull/15)），但维护者倾向于让 OCR 保持轻量 CLI 定位，平台集成放到外部（[PR #11](https://github.com/alibaba/open-code-review/pull/11)）。于是发布能力被拆成独立工具：OCR 负责生成审查结果，`ocr-review-publisher` 负责发布到 GitLab，包含标记管理、摘要控制和 CI 冒烟门禁。

## 当前范围

版本 1 有意保持窄范围：

- OCR 是唯一支持的审查生产者。
- GitLab 是唯一支持的发布平台。
- 专注于评论渲染、安全内联锚点、摘要更新、标记清除和 CI 集成。

不支持 GitHub PR、Webhook、斜杠命令、一键平台建议或严重性/类别门控（除非 OCR 输出提供这些字段）。

## 安装

### 从 GitHub Releases 下载

预编译二进制文件支持 macOS 和 Linux（amd64/arm64）：

```bash
# macOS arm64（Apple Silicon）
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.1/ocr-review-publisher_0.1.1_darwin_arm64.tar.gz
tar xzf ocr-review-publisher_0.1.1_darwin_arm64.tar.gz

# Linux amd64
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.1/ocr-review-publisher_0.1.1_linux_amd64.tar.gz
tar xzf ocr-review-publisher_0.1.1_linux_amd64.tar.gz
```

校验 checksums：

```bash
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.1/ocr-review-publisher_0.1.1_checksums.txt
sha256sum -c ocr-review-publisher_0.1.1_checksums.txt
```

查看所有版本：[GitHub Releases](https://github.com/Fanduzi/ocr-review-publisher/releases)

### 从源码构建

```bash
git clone https://github.com/Fanduzi/ocr-review-publisher.git
cd ocr-review-publisher
make build
```

二进制文件创建在 `./ocr-review-publisher`。

### 验证

```bash
./ocr-review-publisher version
./ocr-review-publisher --help
```

## 快速开始

### 1. 生成 OCR 输出

```bash
ocr review --from origin/main --to HEAD --format json --audience agent > ocr-result.json
```

### 2. 本地渲染（预览）

```bash
./ocr-review-publisher render --input ocr-result.json
```

### 3. 发布到 GitLab

```bash
./ocr-review-publisher publish \
  --platform gitlab \
  --input ocr-result.json \
  --gitlab-base-url https://gitlab.example.com \
  --project-id group/project \
  --mr 123
```

### 4. 清除发布者评论

```bash
./ocr-review-publisher clear \
  --platform gitlab \
  --scope all \
  --gitlab-base-url https://gitlab.example.com \
  --project-id group/project \
  --mr 123
```

## 命令

### `render`

将 OCR 发现渲染为 Markdown 而不发布。用于调试评论质量。

```bash
./ocr-review-publisher render --input ocr-result.json
./ocr-review-publisher render --input ocr-result.json --format json
./ocr-review-publisher render --input - < ocr-result.json
```

### `publish`

将 OCR 发现发布到 GitLab MR。

```bash
./ocr-review-publisher publish \
  --platform gitlab \
  --input ocr-result.json \
  --gitlab-base-url https://gitlab.example.com \
  --project-id group/project \
  --mr 123 \
  --token $GITLAB_TOKEN
```

选项：

- `--dry-run` - 渲染但不发布
- `--no-inline` - 跳过内联评论
- `--no-summary` - 跳过摘要评论
- `--clear-existing` - 先清除发布者评论
- `--format text|json` - 输出格式

### `clear`

清除 GitLab MR 上发布者拥有的评论。

```bash
./ocr-review-publisher clear --platform gitlab --scope inline ...
./ocr-review-publisher clear --platform gitlab --scope summary ...
./ocr-review-publisher clear --platform gitlab --scope all ...
```

### `version`

打印版本信息。

```bash
./ocr-review-publisher version
```

## GitLab 配置

### 环境变量

CLI 从环境变量推断 GitLab 配置：

| 标志 | 环境变量 | 描述 |
|------|---------|------|
| `--token` | `GITLAB_TOKEN` 或 `OCR_GITLAB_TOKEN` | GitLab API 令牌 |
| `--gitlab-base-url` | `OCR_GITLAB_BASE_URL` 或 `CI_SERVER_URL` | GitLab 基础 URL（默认：`https://gitlab.com`） |
| `--project-id` | `CI_PROJECT_ID` | 项目 ID 或命名空间路径 |
| `--mr` | `CI_MERGE_REQUEST_IID` | 合并请求 IID |

标志覆盖环境变量。

### 所需权限

GitLab 令牌需要：

- `api` 范围用于创建/更新/删除笔记
- `read_repository` 范围用于读取 MR 差异

### GitLab 13.12 兼容性

发布者支持 GitLab 13.12 自托管实例：

- 使用 `/diffs` 端点，回退到 `/changes?access_raw_diffs=true`
- 使用 `per_page=100` 和 `X-Next-Page` 头分页讨论

详见 [docs/gitlab.zh-CN.md](docs/gitlab.zh-CN.md) 了解详细 GitLab 用法。

## 安全模型

所有发布者拥有的评论使用稳定标记：

```markdown
<!-- ocr-review-publisher:inline -->
<!-- ocr-review-publisher:summary -->
```

清除操作只删除包含这些标记的笔记。用户评论和其他机器人评论永远不会被触及。

## 开发

```bash
make fmt              # 格式化代码
make test             # 运行测试
make vet              # 运行 go vet
make build            # 构建二进制
make check            # 运行所有检查（fmt + test + vet + build + compat）
make test-compat      # 运行 OCR 输出兼容性测试
make test-e2e-gitlab  # 运行 GitLab e2e 测试（可选，需要环境变量）
make release-readiness # 运行严格发布前门控
```

详见 [CONTRIBUTING.md](CONTRIBUTING.md) 了解开发工作流和 TDD 要求。

## 文档

- [Quality Gates](docs/quality-gates.zh-CN.md) - 完成定义和发布阻止器
- [OCR Compatibility](docs/ocr-compatibility.zh-CN.md) - 解析器兼容性策略和固定装置
- [GitLab Usage](docs/gitlab.zh-CN.md) - 详细 GitLab 配置和故障排除
- [CI Integration](docs/ci.zh-CN.md) - GitLab CI 和 GitHub Actions 示例
- [Output Contract](docs/output-contract.zh-CN.md) - 接受的 OCR 输出格式
- [GitLab E2E Testing](docs/e2e-gitlab.zh-CN.md) - 可选的真实 GitLab 测试
- [Release Process](docs/release.zh-CN.md) - 发布门控和工作流

## 限制

- 仅支持 Open Code Review 输出（不支持其他审查引擎）
- 仅支持 GitLab MR 发布（不支持 GitHub PR）
- 不增强 OCR 的审查智能，只发布其发现
- 类别/严重性徽章仅在 OCR 输出提供这些字段时出现
- 不支持 Webhook/服务器模式
- 不支持一键平台建议
