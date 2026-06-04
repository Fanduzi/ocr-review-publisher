> English version: [smoke-gitlab-ci.md](smoke-gitlab-ci.md)

# GitLab CI 冒烟测试门禁

本文档介绍 `ocr-review-publisher` 的可复现 GitLab CI 冒烟测试门禁。

## 概述

CI 冒烟测试门禁验证发布器在真实 GitLab CI 流水线中是否正常工作。它会：

1. 在测试项目中创建或更新专用冒烟分支，写入 `.gitlab-ci.yml`
2. 通过 GitLab API 触发流水线
3. 等待流水线完成
4. 验证 OCR 运行、发布器运行、MR 评论符合质量断言
5. 可选清理标记评论

默认保留冒烟分支和 `.gitlab-ci.yml`，以便在 GitLab 实例上检查 CI 配置和流水线历史。

## 前置条件

- 运行中的 GitLab 实例和测试项目
- 已注册并可从 GitLab 实例访问的 Docker runner
- OCR LLM 凭据（`OCR_LLM_URL`、`OCR_LLM_TOKEN`、`OCR_LLM_MODEL`）
- 测试项目中有一个打开的合并请求

## 使用方式

```bash
# 运行并清理（默认）：验证后删除评论
make smoke-gitlab-ci

# 运行不清理：保留评论供人工检查
OCR_CI_SMOKE_CLEANUP=0 make smoke-gitlab-ci
```

## 环境变量

必需（通常在 `env.gitlab.local` 中，由 Makefile target 自动 source）：

| 变量 | 描述 |
|------|------|
| `OCR_E2E_GITLAB_URL` | GitLab 基础 URL |
| `OCR_E2E_GITLAB_TOKEN` | GitLab API 令牌 |
| `OCR_E2E_GITLAB_PROJECT_ID` | 项目 ID 或命名空间路径 |
| `OCR_E2E_GITLAB_MR_IID` | 合并请求 IID |
| `OCR_LLM_URL` | OCR 的 LLM API 端点 |
| `OCR_LLM_TOKEN` | LLM API 令牌 |
| `OCR_LLM_MODEL` | LLM 模型名称 |

可选：

| 变量 | 默认值 | 描述 |
|------|--------|------|
| `OCR_CI_SMOKE_BRANCH` | `ci-smoke/ocr-review-publisher` | 冒烟分支名 |
| `OCR_CI_SMOKE_CLEANUP` | `1` | 设为 `0` 保留评论 |
| `OCR_CI_SMOKE_KEEP_COMMENTS` | `0` | 别名：设为 `1` 不清理 |
| `OCR_CI_SMOKE_TIMEOUT` | `900` | 流水线超时（秒） |
| `OCR_CI_SMOKE_POLL` | `5` | 轮询间隔（秒） |

## 工作原理

### 冒烟分支

脚本从 MR 的源分支创建名为 `ci-smoke/ocr-review-publisher`（可配置）的分支，通过 GitLab API 写入 `.gitlab-ci.yml`。每次运行时重置分支以匹配最新的 MR 源 HEAD。

CI 配置安装 Node.js、Go、OCR CLI，并从源码构建发布器。然后对 MR 运行 `ocr review` 和 `ocr-review-publisher publish`。

### 流水线触发

流水线通过 GitLab API 触发（非合并请求事件），所有必需变量作为流水线变量显式传递。包括发布器的 `OCR_GITLAB_TOKEN`、OCR 的 `OCR_LLM_URL`/`OCR_LLM_TOKEN`/`OCR_LLM_MODEL`，以及作业的 `CI_GITLAB_URL`/`CI_PROJECT_ID`/`CI_MR_IID`。这样就不依赖项目 CI/CD 变量或 `CI_MERGE_REQUEST_IID`（API 触发的流水线不会设置该变量）。

### URL 报告

脚本使用 GitLab API 响应中的 `web_url` 字段（流水线、作业、MR）报告所有 URL。脚本中没有硬编码项目路径。

### 质量断言

流水线成功后，脚本获取 MR 讨论并断言：

- 恰好存在 1 个摘要标记评论
- 存在内联标记评论（除非 OCR 没有找到可内联发布的发现）
- 没有 `line_code can't be blank` 错误
- 没有 `suggestion` 围栏
- Go 文件评论使用 ` ```go ` 语言围栏
- 没有重复的摘要标记

### 清理行为

默认（`OCR_CI_SMOKE_CLEANUP=1`）情况下，脚本在验证后删除所有发布器拥有的标记评论，并断言计数返回 0/0。

设置 `OCR_CI_SMOKE_CLEANUP=0` 时，评论保留在 MR 上供人工检查。脚本会打印 MR URL 以便查看。

## Runner 基础设施

冒烟测试门禁需要一个注册到 GitLab 实例的 Docker runner。Runner 配置和令牌位于本地配置目录，不应提交。

检查 runner 状态：

```bash
docker ps --filter name=gitlab-runner
```

移除 runner（如需要）：

```bash
docker stop gitlab-runner && docker rm gitlab-runner
```

## 输出

脚本输出：

- 流水线 URL 和作业 URL
- 冒烟分支名
- 清理模式
- 预清除、发布后、清理后的标记计数
- 最终 PASSED/FAILED 状态
