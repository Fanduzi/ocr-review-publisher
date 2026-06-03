# ocr-review-publisher v0.1.0 发布说明

## 概览

`ocr-review-publisher` 首次发布。它将 [Open Code Review (OCR)](https://github.com/alibaba/open-code-review) 的审查结果发布为 GitLab MR 评论。

## 亮点

- 将 OCR 审查输出发布为 GitLab MR 行内讨论和摘要评论
- 基于稳定标记的所有权管理：发布器只管理自己创建的评论
- 渲染结果使用语言感知的代码围栏
- 可复现的 CI 冒烟测试门禁，用于持续质量验证
- 跨平台发布资产（macOS 和 Linux，amd64 和 arm64）

## 新增内容

### GitLab MR 发布

- 解析 OCR `--format json --audience agent` 输出（文件或标准输入）
- 为带安全 diff 锚点的发现创建行内讨论
- 创建和更新摘要评论，包含发现计数和诊断信息
- 清理发布器拥有的评论，不影响用户或其他机器人评论
- 单个行内评论失败时继续发布

### Markdown 渲染

- 语言感知的代码围栏（Go、JavaScript、TypeScript、Python、Java、Rust、JSON、YAML、Markdown）
- 折叠详情块用于审查上下文和发布诊断
- 当 OCR 输出包含分类和严重性字段时显示对应徽章
- 稳定的所有权标记，用于生命周期管理

### CI 集成

- GitLab CI 环境变量推断（`CI_SERVER_URL`、`CI_PROJECT_ID`、`CI_MERGE_REQUEST_IID`）
- `--dry-run` 模式用于预览
- `--format json` 输出机器可读报告
- `--fail-on-publish-error` 用于严格 CI

### 质量门禁

- 所有评论类型的黄金 Markdown 测试
- 真实 OCR 固定测试，覆盖代表性审查输出
- 使用 `httptest` 的 GitLab API 形状测试
- diff 锚点选择测试，覆盖边缘情况
- 可选的 GitLab 13.12 端到端测试
- 真实 OCR 本地冒烟测试门禁
- 可复现的 GitLab CI 冒烟测试门禁

## 安装

从 GitHub Releases 下载：

```bash
# Linux amd64
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.0/ocr-review-publisher_0.1.0_linux_amd64.tar.gz
tar xzf ocr-review-publisher_0.1.0_linux_amd64.tar.gz

# macOS arm64（Apple Silicon）
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.0/ocr-review-publisher_0.1.0_darwin_arm64.tar.gz
tar xzf ocr-review-publisher_0.1.0_darwin_arm64.tar.gz

# macOS amd64（Intel）
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.0/ocr-review-publisher_0.1.0_darwin_amd64.tar.gz
tar xzf ocr-review-publisher_0.1.0_darwin_amd64.tar.gz
```

校验 checksums：

```bash
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.0/ocr-review-publisher_0.1.0_checksums.txt
sha256sum -c ocr-review-publisher_0.1.0_checksums.txt
```

或从源码构建：

```bash
git clone https://github.com/Fanduzi/ocr-review-publisher.git
cd ocr-review-publisher
make build
```

## 兼容性

- 已验证 OCR 版本：v1.1.9
- 支持平台：GitLab（包括 13.12 自托管）
- 支持操作系统：darwin、linux
- 支持架构：amd64、arm64

## 质量门禁

- `make check`（fmt、test、vet、build、test-compat）：通过
- `make test-e2e-gitlab`：通过（GitLab 13.12）
- `make smoke-gitlab-real-ocr`：通过（真实 OCR v1.1.9 对固定仓库测试）
- `make smoke-gitlab-ci`：通过（可复现 CI 冒烟测试门禁）
- `make release-readiness`：通过
- `make release-snapshot`：通过（darwin/linux amd64/arm64 归档及校验和）

## 已知限制

- 仅支持 GitLab：本版本不支持 GitHub PR 发布
- 无 webhook、服务器或斜杠命令模式
- 无 Homebrew、npm 或 Docker 包（仅单二进制分发）
- 仅当 OCR 输出包含分类和严重性字段时才渲染对应徽章
- 未启用真正的一键 GitLab 建议；建议以普通代码围栏渲染
- 需要单独安装 OCR CLI 以生成审查输入

## 升级说明

- 首次发布；无需迁移
- 发布器使用稳定标记（`<!-- ocr-review-publisher:inline -->`、`<!-- ocr-review-publisher:summary -->`）进行评论所有权管理
