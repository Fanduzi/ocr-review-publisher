> English version: [output-contract.md](output-contract.md)

# 输出契约

本文档描述 `ocr-review-publisher` 接受的 [Open Code Review (OCR)](https://github.com/alibaba/open-code-review) 输出格式。

## 接受的格式

解析器接受以下命令的输出：

```bash
ocr review --format json --audience agent
```

## JSON 结构

```json
{
  "status": "success",
  "message": "Review completed",
  "comments": [
    {
      "path": "service/user.go",
      "content": "Error return value is not checked",
      "existing_code": "fmt.Println(err)",
      "suggestion_code": "if err != nil {\n\treturn err\n}",
      "start_line": 37,
      "end_line": 37,
      "thinking": "The error is silently discarded."
    }
  ],
  "warnings": [
    {
      "type": "subtask_failed",
      "path": "service/user.go",
      "message": "Could not analyze file"
    }
  ]
}
```

## 字段说明

### 顶层

| 字段 | 类型 | 必需 | 描述 |
|------|------|------|------|
| `status` | string | 否 | 审查状态（如 "success"、"partial"） |
| `message` | string | 否 | 人类可读的状态消息 |
| `comments` | array | 否 | 发现对象数组 |
| `warnings` | array | 否 | 警告对象数组 |

如果 `comments` 缺失或为 null，解析器返回空的发现切片。

### Comment（发现）

| 字段 | 类型 | 必需 | 描述 |
|------|------|------|------|
| `path` | string | 是 | 相对于仓库根目录的文件路径 |
| `content` | string | 是 | 发现描述 |
| `existing_code` | string | 否 | 文件中的代码片段 |
| `suggestion_code` | string | 否 | 建议的替换代码 |
| `start_line` | int | 否 | 起始行号（未知时为 0） |
| `end_line` | int | 否 | 结束行号（未知时为 0） |
| `thinking` | string | 否 | 审查者推理/上下文 |
| `category` | string | 否 | 发现类别（如 "security"、"performance"） |
| `severity` | string | 否 | 发现严重性（如 "high"、"medium"、"low"） |

### Warning（警告）

| 字段 | 类型 | 必需 | 描述 |
|------|------|------|------|
| `type` | string | 是 | 警告类型（如 "subtask_failed"、"timeout"） |
| `path` | string | 否 | 关联的文件路径 |
| `message` | string | 是 | 警告描述 |

## 带前缀的输出

OCR 可能在 JSON 对象前打印人类可读文本：

```
Review completed in 12.3s using model claude-sonnet-4-20250514
Found 1 finding across 1 file.
{
  "status": "success",
  "comments": [...]
}
```

解析器通过找到第一个 `{` 字符并从那里开始解析来处理这种情况。

## 可选字段

### 前向兼容

未知的顶层字段保存在 `Result.Metadata` 中。

未知的 comment 字段保存在 `Finding.Metadata` 中。

未知的 warning 字段保存在 `Warning.Metadata` 中。

### 类别和严重性

当 OCR 输出包含 `category` 和 `severity` 字段时，它们映射到内部模型的强类型字段。缺失时默认为空字符串。

### 置信度

当 OCR 输出包含 `confidence` 字段时，它保存在 `Finding.Metadata["confidence"]` 中。

## 行号

- 缺失的行号默认为 0，而非失败
- 零行号是有效的（用于摘要回退）
- 负行号按原样保留

## 错误处理

### 空输入

返回错误："empty input"

### 无 JSON 对象

返回错误："no JSON object found in input"

### 格式错误的 JSON

返回带上下文的错误："failed to parse JSON: ..."

错误消息简洁，不会输出大量输入摘录。

## 兼容性策略

解析器应支持：

- 本项目文档记录的最低 OCR 版本
- 定时 CI 验证的最新 OCR 版本
- 前向兼容的可选字段

解析器应对格式错误的 JSON 严格，但对 JSON 对象前的无害包装文本保持宽容。

## 固定

兼容性固定存储在 `testdata/ocr/` 下：

- `basic.json` - 含发现的标准输出
- `prefixed-agent-output.txt` - 带前导文本的输出
- `empty-comments.json` - 空评论的输出
- `with-warnings.json` - 含警告的输出
- `future-fields.json` - 含 category、severity 及未知字段的输出

固定不得包含密钥、私有仓库名、私有 URL 或本地文件系统路径。
