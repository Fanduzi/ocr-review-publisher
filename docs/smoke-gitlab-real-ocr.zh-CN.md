> English version: [smoke-gitlab-real-ocr.md](smoke-gitlab-real-ocr.md)

# 真实 OCR 冒烟测试门禁

本文档介绍真实 OCR 冒烟测试门禁，验证从 OCR 输出到 GitLab MR 评论的完整流水线。

## 概述

真实 OCR 冒烟测试门禁（`make smoke-gitlab-real-ocr`）测试完整工作流：

1. **预清除** MR 上所有发布器拥有的评论
2. **验证** 预清除后标记计数为零
3. 对固定仓库运行真实 **OCR 审查**
4. 解析 OCR 输出并**发布**到 GitLab MR
5. 通过 GitLab API **验证**评论质量（仅本次运行的标记）
6. **清理**发布器拥有的评论（当 `OCR_SMOKE_CLEANUP=1` 时）
7. **验证** 清理后标记计数为零

这是一个**仅限维护者**的冒烟测试门禁，需要：
- 有真实代码变更的本地固定仓库
- 配置了 LLM 凭据的 OCR 二进制
- 具有 API 访问权限的 GitLab 实例
- 有效的 GitLab 令牌和 MR

## 快速开始

```bash
# 构建发布器二进制
make build

# 运行冒烟测试门禁
OCR_SMOKE_REPO=~/path/to/fixture make smoke-gitlab-real-ocr
```

## 必需环境变量

### GitLab 配置

```bash
export OCR_E2E_GITLAB_URL=https://your-gitlab.example.com
export OCR_E2E_GITLAB_TOKEN=your-gitlab-token
export OCR_E2E_GITLAB_PROJECT_ID=123
export OCR_E2E_GITLAB_MR_IID=456
```

### 固定仓库

```bash
export OCR_SMOKE_REPO=~/path/to/fixture-repo
```

固定仓库应具备：
- `main` 分支上至少有一个提交
- 最近的变更（在 `HEAD` 或功能分支上）供 OCR 审查
- 推荐包含 Go 文件以测试语言感知围栏

## 可选环境变量

| 变量 | 默认值 | 描述 |
|------|--------|------|
| `OCR_SMOKE_OCR_BIN` | `ocr` | OCR 二进制路径（回退到 npx） |
| `OCR_SMOKE_PUBLISHER_BIN` | `./ocr-review-publisher` | 发布器二进制路径 |
| `OCR_SMOKE_FROM` | `main` | OCR 审查的基准 ref |
| `OCR_SMOKE_TO` | `HEAD` | OCR 审查的头 ref |
| `OCR_SMOKE_CONCURRENCY` | `2` | 并发设置 |
| `OCR_SMOKE_TIMEOUT` | `15` | 超时时间（分钟） |
| `OCR_SMOKE_CLEANUP` | `1` | 运行后清理评论 |
| `OCR_SMOKE_KEEP_OUTPUT` | `0` | 在 `.local/smoke/` 中保留输出文件 |

## 本地环境文件

本地开发时，在仓库根目录创建 `env.gitlab.local`：

```bash
cat > env.gitlab.local <<'EOF'
OCR_E2E_GITLAB=1
OCR_E2E_GITLAB_URL=https://your-gitlab.example.com
OCR_E2E_GITLAB_TOKEN=your-token
OCR_E2E_GITLAB_PROJECT_ID=123
OCR_E2E_GITLAB_MR_IID=456
EOF
```

此文件被 git 忽略，不应提交。

当 `env.gitlab.local` 存在时，`make smoke-gitlab-real-ocr` 会自动 source 它。

## 质量断言

冒烟测试门禁在清理前验证评论质量：

### 必需断言

1. **OCR 标记评论存在** - 至少 1 个带有 OCR 标记的内联或摘要评论
2. **摘要标记存在** - 至少 1 个摘要标记评论
3. **无原始 API 错误** - 没有 `line_code can't be blank` 错误
4. **无建议围栏** - 没有 ` ```suggestion ` 围栏（使用语言感知围栏）
5. **语言感知围栏** - Go 文件使用 ` ```go ` 围栏
6. **代码在围栏块中** - 已有代码在围栏代码块中
7. **审查上下文块** - 存在 `<details><summary>Review context</summary>`
8. **无重复摘要** - 仅 1 个摘要笔记（多次发布运行时）

### 可选断言

- **诊断块** - 存在警告时显示 `<details><summary>Publish diagnostics`

## 输出文件

默认情况下，输出文件保存到临时目录，退出时删除。

要保留输出文件用于调试：

```bash
OCR_SMOKE_KEEP_OUTPUT=1 OCR_SMOKE_REPO=~/path/to/fixture make smoke-gitlab-real-ocr
```

输出文件将保存到 `.local/smoke/`：
- `ocr-output.json` - 原始 OCR 输出
- `ocr-output-clean.json` - 清理后的 OCR 输出（不含摘要行）
- `ocr-output-final.json` - 最终 OCR 输出
- `publish-output.txt` - 发布器输出
- `discussions.json` - GitLab 讨论 API 响应
- `notes.json` - 提取的笔记
- `inline-notes.json` - 内联标记笔记
- `summary-notes.json` - 摘要标记笔记
- `verification-results.json` - 验证结果

## 清理行为

冒烟测试门禁在发布前始终**预清除**已有的发布器拥有的评论。这确保质量断言仅验证当前运行的评论，而非历史评论。无论 `OCR_SMOKE_CLEANUP` 设置如何，预清除步骤都会运行。

验证后，冒烟测试门禁默认**清理**所有发布器拥有的评论（`OCR_SMOKE_CLEANUP=1`）。这会删除当前运行创建的所有评论。

跳过最终清理（用于调试）：

```bash
OCR_SMOKE_CLEANUP=0 OCR_SMOKE_REPO=~/path/to/fixture make smoke-gitlab-real-ocr
```

**注意：** 设置 `OCR_SMOKE_CLEANUP=0` 仅跳过最终清理。预清除步骤始终运行以确保干净的测试环境。设置 `OCR_SMOKE_CLEANUP=0` 后，当前运行的评论将保留在 MR 上供人工检查。

## 故障排除

### OCR 二进制未找到

如果 OCR 未全局安装，脚本会尝试 `npx @alibaba-group/open-code-review`。

全局安装 OCR：
```bash
npm install -g @alibaba-group/open-code-review
```

### 发布器二进制未找到

先构建发布器：
```bash
make build
```

### 固定仓库未找到

确保 `OCR_SMOKE_REPO` 指向有效的 git 仓库：
```bash
ls -la ~/path/to/fixture/.git
```

### GitLab API 错误

检查 GitLab 令牌和权限：
```bash
curl -H "PRIVATE-TOKEN: $OCR_E2E_GITLAB_TOKEN" \
  "$OCR_E2E_GITLAB_URL/api/v4/projects/$OCR_E2E_GITLAB_PROJECT_ID"
```

### OCR 审查失败

确保 OCR 配置了 LLM 凭据。参见 OCR 文档了解支持的提供商。

## 与 `make test-e2e-gitlab` 的对比

| 特性 | `make test-e2e-gitlab` | `make smoke-gitlab-real-ocr` |
|------|------------------------|------------------------------|
| **输入** | 构造的 `review.Result` | 真实 OCR 输出 |
| **需要 OCR** | 否 | 是 |
| **需要 LLM** | 否 | 是 |
| **需要固定仓库** | 否 | 是 |
| **CI 默认** | 是（需要环境变量） | 否 |
| **目的** | 发布器正确性 | 完整流水线验证 |

## 安全说明

- 冒烟测试门禁在指定 MR 上创建和删除评论
- 除非了解影响，否则不要对生产 MR 运行
- 脚本默认会自行清理
- 输出文件可能包含敏感数据（OCR 发现、API 响应）
- 本地环境文件包含密钥，不应提交

## CI 集成

此冒烟测试门禁默认**不**用于 CI。它需要：
- 配置了 LLM 凭据的真实 OCR 二进制
- 固定仓库访问权限
- 具有写入权限的 GitLab API 访问

CI 中请使用 `make test-e2e-gitlab` 配合构造的测试数据。

## 相关文档

- [GitLab E2E 测试](e2e-gitlab.zh-CN.md) - 使用构造数据的发布器 e2e 测试
- [GitLab 使用指南](gitlab.zh-CN.md) - GitLab 平台文档
- [OCR 兼容性](ocr-compatibility.zh-CN.md) - OCR 输出格式兼容性
