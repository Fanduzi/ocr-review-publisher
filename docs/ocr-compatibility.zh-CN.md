> English version: [ocr-compatibility.md](ocr-compatibility.md)

# OCR 输出兼容性

`ocr-review-publisher` 依赖 `ocr review --format json --audience agent` 的机器可读输出。该输出是 [Open Code Review](https://github.com/alibaba/open-code-review) 拥有的外部契约，因此本项目必须持续验证它。

## 兼容性策略

解析器应支持：

- 本项目文档记录的最低 OCR 版本
- 通过手动实时捕获（`workflow_dispatch`）验证的最新 OCR 版本
- 前向兼容的可选字段，如 `category`、`severity`、`confidence`

解析器应对格式错误的 JSON 严格，但对 JSON 对象前的无害包装文本保持宽容，因为某些 OCR 模式可能在结构化负载前打印摘要行。

## 固定策略

将捕获的 OCR 输出保存在 `testdata/ocr/` 或 `testdata/fixtures/` 下。

每个固定应记录：

- OCR 版本
- 使用的命令
- 输出来自实时 LLM 运行还是清理样本
- 预期的解析结果形状

固定不得包含密钥、私有仓库名、私有 URL 或本地文件系统路径。

## 本地兼容性流程

本地运行兼容性测试：

```bash
make test-compat
```

这运行 `go test ./internal/compat -count=1`，验证所有已签入的固定正确解析并具有预期形状。不需要网络或 LLM 凭据。

从实时 OCR 运行重新生成固定：

```bash
scripts/capture-ocr-output.sh --ocr-version latest --output testdata/ocr/latest.json
```

捕获脚本创建临时样本仓库（含确定性变更），安装指定的 OCR 版本，运行 `ocr review --format json --audience agent`，并将输出写入指定路径。OCR 生成审查需要配置 LLM 凭据。

## 已签入固定

固定存储在 `testdata/ocr/` 下：

- `basic.json` - 标准 OCR 输出，含发现、已有代码、建议、思考
- `prefixed-agent-output.txt` - JSON 对象前有非 JSON 文本的 OCR 输出
- `empty-comments.json` - 空评论数组的 OCR 输出
- `with-warnings.json` - 含警告的 OCR 输出
- `future-fields.json` - 含 category、severity、confidence 及未知未来字段的 OCR 输出
- `ocr-v1.1.13-live.json` - OCR v1.1.13 实时运行的清理后真实输出

固定不得包含本地路径、令牌、私有 URL 或真实 GitLab 信息。

## GitHub Actions

项目使用两级 CI：

- **PR CI**（`.github/workflows/ci.yml`）：运行 `make check`，包含 `make test-compat`、单元测试、vet、构建和格式检查。
- **OCR 兼容性 CI**（`.github/workflows/ocr-compatibility.yml`）：
  - **定时**（每周）：仅运行已签入的固定兼容性测试。不需要 LLM 凭据。
  - **手动**（`workflow_dispatch`）：运行固定测试和最新 OCR 实时捕获。需要 LLM 密钥。缺少密钥时显式失败。

定时兼容性 CI 不需要 GitLab 令牌或平台访问权限。实时捕获仅限手动，需要配置 GitHub Actions 密钥。

### 所需 GitHub Actions 密钥和变量

实时捕获需要以下密钥（设置 > 密钥和变量 > 操作）：

| 名称 | 类型 | 描述 |
|------|------|------|
| `OCR_LLM_URL` | 密钥 | LLM API 端点 URL |
| `OCR_LLM_TOKEN` | 密钥 | LLM API 令牌 |
| `OCR_LLM_MODEL` | 密钥 | LLM 模型名称 |
| `OCR_USE_ANTHROPIC` | 变量 | 可选。设为 `true` 使用 Anthropic 协议 |

这些名称与 Open Code Review LLM 解析器契约一致（已对照 `internal/llm/resolver.go` 验证）。解析器也支持 Claude Code 环境变量（`ANTHROPIC_BASE_URL`、`ANTHROPIC_AUTH_TOKEN`、`ANTHROPIC_MODEL`）作为备选。

### 实时捕获直接验证

实时捕获运行时，工作流使用 `TestCapturedOCROutputParses` 直接解析 `/tmp/ocr-latest.json`。这验证捕获的输出是有效的 OCR JSON，而非仅验证已签入的固定仍能解析。

## 发布要求

公开发布前：

- `make test-compat` 必须通过
- 最新 OCR 兼容性工作流必须为绿色，或在发布前记录已知不兼容性
- 发布说明必须注明已验证的 OCR 版本范围
