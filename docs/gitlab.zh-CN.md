> English version: [gitlab.md](gitlab.md)

# GitLab 使用指南

本文档介绍 `ocr-review-publisher` 的 GitLab 配置和使用细节。

## 令牌要求

### 创建令牌

1. 进入 GitLab > 用户设置 > 访问令牌
2. 创建具有 `api` 范围的令牌
3. 对于自托管 GitLab，确保令牌有权访问目标项目

### 所需权限

- **api** - 创建、更新和删除合并请求笔记
- **read_repository** - 读取合并请求差异和版本

### CI/CD 令牌

在 GitLab CI 中，将具有 `api` 范围的个人访问令牌、项目访问令牌或组访问令牌存储为 CI/CD 变量：

```yaml
variables:
  GITLAB_TOKEN: ${OCR_GITLAB_TOKEN}
```

> **注意：** 内置的 `CI_JOB_TOKEN` 权限有限，在部分 GitLab 版本上不支持创建合并请求讨论。使用具有 `api` 范围的专用令牌以确保发布可靠。

## 项目标识

### 项目 ID

使用数字项目 ID：

```bash
--project-id 123
```

在 GitLab > 项目 > 设置 > 通用 中查找项目 ID。

### 命名空间路径

使用 URL 编码的命名空间路径：

```bash
--project-id group/subgroup/project
```

发布器会自动进行 URL 编码（例如 `group%2Fsubgroup%2Fproject`）。

## 合并请求 IID

MR IID（内部 ID）是 MR URL 中显示的数字：

```
https://gitlab.example.com/group/project/-/merge_requests/42
                                                  ^^
                                                  MR IID = 42
```

这与全局 MR ID 不同。

## 自托管 GitLab

对于自托管 GitLab 实例：

```bash
./ocr-review-publisher publish \
  --platform gitlab \
  --gitlab-base-url https://gitlab.internal.example.com \
  --project-id group/project \
  --mr 42 \
  --input ocr-result.json
```

或通过环境变量：

```bash
export OCR_GITLAB_BASE_URL=https://gitlab.internal.example.com
```

### GitLab 13.12 兼容性

发布器支持 GitLab 13.12 自托管实例：

- 使用 `/diffs` 端点获取变更文件
- 当 `/diffs` 返回 404 时回退到 `/changes?access_raw_diffs=true`
- 使用 `per_page=100` 和精确的 `X-Next-Page` 头分页讨论

## 内联评论

### 锚点工作原理

发布器在差异中的新增行上创建内联评论：

1. 获取变更文件及其差异
2. 获取差异版本 SHA 用于位置引用
3. 对于每个发现，选择范围内第一个新增行
4. 在该位置创建内联讨论

### 跳过评论的情况

以下情况会跳过内联评论（包含在摘要诊断中）：

- 发现没有路径或没有正数行范围
- 发现范围内没有新增行
- 差异不可用或无法解析
- 差异版本缺失
- GitLab 拒绝内联位置

即使跳过内联，所有发现仍会包含在摘要中。

## 摘要评论

### 创建行为

当不存在已有的摘要标记笔记时，发布器创建新笔记，包含：

- 产品头部
- 发现总数
- 带路径和行范围的发现列表
- 跳过/失败的内联评论诊断
- 摘要标记

### 更新行为

当存在已有的摘要标记笔记时，发布器更新它而非创建重复项。这确保重复的发布运行不会创建多个摘要。

## 清除行为

### 范围选项

- `--scope inline` - 仅删除内联标记笔记
- `--scope summary` - 仅删除摘要标记笔记
- `--scope all` - 删除内联和摘要标记笔记

### 安全性

清除操作仅删除包含发布器标记的笔记。用户评论和其他机器人评论永远不会被删除。

## 故障排除

### 无效的行锚点

**症状：** 内联评论被跳过，提示 "no added line in range"

**原因：** 发现指向的行在差异中是上下文行（非新增行）。

**修复：** 确保发现指向 MR 中实际新增或修改的行。

### 无差异版本

**症状：** 所有内联评论失败，提示 "no GitLab diff version available"

**原因：** MR 没有差异版本（可能发生在非常旧或空的 MR 上）。

**修复：** 确保 MR 有实际变更。摘要仍会被创建。

### 令牌权限

**症状：** 403 Forbidden 错误

**原因：** 令牌缺少所需权限。

**修复：** 确保令牌具有 `api` 范围并有权访问目标项目。

### 自托管基础 URL

**症状：** 连接被拒绝或 DNS 错误

**原因：** 基础 URL 不正确。

**修复：** 确保基础 URL 包含协议（https://）且不以斜杠结尾。

### 评论被跳过但摘要已创建

**症状：** 摘要存在但没有内联评论

**原因：** 发现未指向差异中的新增行，或差异不可用。

**修复：** 这是预期行为。查看摘要诊断了解哪些发现被跳过及原因。
