# ocr-review-publisher v0.1.1 发布说明

## 概览

v0.1.0 之后的维护版本。修复发布正文发布、OCR 兼容性实时捕获，并补充中文文档。

## 变更内容

### 发布基础设施

- 修复发布工作流，通过 GoReleaser 之后的 `gh release edit` 发布双语发布说明正文
- 更新 GitHub 仓库描述和主题

### OCR 兼容性

- 修复实时捕获环境变量契约：使用 `OCR_LLM_URL`、`OCR_LLM_TOKEN`、`OCR_LLM_MODEL`（已对照 Open Code Review 解析器验证）
- 添加通过 `TestCapturedOCROutputParses` 直接解析捕获输出
- 添加清理后的真实 OCR 固定（`ocr-v1.1.13-live.json`）以覆盖解析器回归
- 手动实时捕获在缺少密钥时显式失败

### 文档

- 为所有公开文档补充中文版本
- 添加外部 CLI 契约质量门禁
- `.gitignore` 中忽略本地 IDE 文件

## 安装

从 GitHub Releases 下载：

```bash
# Linux amd64
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.1/ocr-review-publisher_0.1.1_linux_amd64.tar.gz
tar xzf ocr-review-publisher_0.1.1_linux_amd64.tar.gz

# macOS arm64（Apple Silicon）
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.1/ocr-review-publisher_0.1.1_darwin_arm64.tar.gz
tar xzf ocr-review-publisher_0.1.1_darwin_arm64.tar.gz
```

校验 checksums：

```bash
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.1/ocr-review-publisher_0.1.1_checksums.txt
sha256sum -c ocr-review-publisher_0.1.1_checksums.txt
```

## 兼容性

- 已验证 OCR 版本：v1.1.13
- 支持平台：GitLab（包括 13.12 自托管）
- 支持操作系统：darwin、linux
- 支持架构：amd64、arm64

## 已知限制

- 仅支持 GitLab：不支持 GitHub PR 发布
- 无 webhook、服务器或斜杠命令模式
- 无 Homebrew、npm 或 Docker 包
- 仅当 OCR 输出包含类别和严重性字段时才渲染对应徽章
- 未启用真正的一键 GitLab 建议

## 升级说明

- 与 v0.1.0 无破坏性变更
- 无标记格式变更
- 无 GitLab 发布行为变更
