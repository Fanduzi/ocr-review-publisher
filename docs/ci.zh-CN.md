> English version: [ci.md](ci.md)

# CI 集成

本文档介绍 `ocr-review-publisher` 的 CI/CD 集成方式。

## GitLab CI

在 GitLab CI 流水线中使用发布器，将 [Open Code Review (OCR)](https://github.com/alibaba/open-code-review) 的审查结果发布到合并请求。

### 基本示例

下面的配置安装 Node.js（OCR CLI 依赖）和 Go，然后从源码构建发布器。`golang` 镜像的默认包仓库不包含 Node.js，因此示例中下载了预编译的 Node.js 二进制。

```yaml
stages:
  - review

image: node:22-bookworm

variables:
  GIT_DEPTH: "0"

review:
  stage: review
  before_script:
    - curl -fsSL https://go.dev/dl/go1.26.1.linux-$(dpkg --print-architecture).tar.gz | tar -xz -C /usr/local
    - export PATH=$PATH:/usr/local/go/bin
    - npm install -g @alibaba-group/open-code-review
    - git clone --depth=1 https://github.com/Fanduzi/ocr-review-publisher.git /tmp/ocr-publisher
    - cd /tmp/ocr-publisher && go build -o /usr/local/bin/ocr-review-publisher ./cmd/ocr-review-publisher
    - cd "$CI_PROJECT_DIR"
  script:
    - export PATH=$PATH:/usr/local/go/bin
    - |
      ocr review --from origin/main --to HEAD --format json --audience agent > ocr-result.json
    - |
      ocr-review-publisher publish \
        --platform gitlab \
        --input ocr-result.json \
        --format text
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
```

> **令牌说明：** 将具有 `api` 范围的个人访问令牌、项目访问令牌或组访问令牌配置为 CI/CD 变量 `GITLAB_TOKEN`。内置的 `CI_JOB_TOKEN` 权限有限，在部分 GitLab 版本上不支持创建合并请求讨论。发布器会自动从环境变量读取 `GITLAB_TOKEN`。

### 发布前先清除

```yaml
review:
  stage: review
  script:
    - |
      ocr-review-publisher clear --platform gitlab --scope all || true
    - |
      ocr review --from origin/main --to HEAD --format json --audience agent > ocr-result.json
    - |
      ocr-review-publisher publish \
        --platform gitlab \
        --input ocr-result.json
```

### 预览模式

使用 `--dry-run` 仅渲染不发布：

```yaml
review-dry-run:
  stage: review
  script:
    - |
      ./ocr-review-publisher publish \
        --platform gitlab \
        --input ocr-result.json \
        --dry-run
```

### JSON 输出

使用 `--format json` 获取机器可读的输出：

```yaml
review:
  stage: review
  script:
    - |
      ./ocr-review-publisher publish \
        --platform gitlab \
        --input ocr-result.json \
        --format json > publish-report.json
  artifacts:
    paths:
      - publish-report.json
```

## 必需环境变量

### GitLab 内置变量

以下变量在合并请求流水线中由 GitLab CI 自动设置：

| 变量 | 描述 | 来源 |
|------|------|------|
| `CI_PROJECT_ID` | 项目 ID | GitLab CI 内置变量 |
| `CI_MERGE_REQUEST_IID` | MR IID | GitLab CI 内置变量 |
| `CI_SERVER_URL` | GitLab URL | GitLab CI 内置变量 |

### 需要配置的 CI/CD 变量

在 **设置 > CI/CD > 变量** 中配置：

| 变量 | 描述 | 脱敏 |
|------|------|------|
| `GITLAB_TOKEN` | 具有 `api` 范围的 GitLab API 令牌 | 是 |

### OCR LLM 变量

OCR CLI 需要 LLM 凭据才能执行代码审查。将以下变量配置为 CI/CD 变量：

| 变量 | 描述 | 脱敏 |
|------|------|------|
| `OCR_LLM_URL` | LLM API 端点 URL | 否 |
| `OCR_LLM_TOKEN` | LLM API 令牌 | 是 |
| `OCR_LLM_MODEL` | LLM 模型名称 | 否 |

## GitHub Actions（本项目）

本节记录 `ocr-review-publisher` 项目自身使用的 GitHub Actions 工作流。这些工作流用于发布器的 CI/CD，而非发布到 GitHub PR（当前不支持）。

### PR CI

项目使用 GitHub Actions 进行自身 CI：

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26.1'
      - run: make check
```

### OCR 兼容性

每周定时检查 OCR 输出兼容性：

```yaml
name: OCR Compatibility
on:
  schedule:
    - cron: '23 3 * * 1'
  workflow_dispatch:
```

定时任务运行固定兼容性测试，不需要密钥。手动触发时，如配置了 LLM 凭据则运行实时捕获。

## CI 失败条件

### 当前行为

- 解析器兼容性失败
- 构建/测试/vet 失败
- 格式检查失败

### 后续行为（待实现）

- 渲染评论质量失败
- 严重性/类别门控失败（当 OCR 提供这些字段时）

## 输出模式

### 文本模式（默认）

人类可读的输出：

```
Published: 3 inline, skipped 1, failed 0
Summary: created
```

### JSON 模式

机器可读的输出：

```json
{
  "inline_published": 3,
  "inline_skipped": 1,
  "inline_failed": 0,
  "summary_created": true,
  "summary_updated": false
}
```

JSON 模式仅将 JSON 输出到 stdout。诊断信息和错误输出到 stderr。
