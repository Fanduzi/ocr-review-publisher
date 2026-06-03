# ocr-review-publisher vX.Y.Z 发布说明

## 概览

本次发布简要摘要。

## 亮点

- 关键变更 1
- 关键变更 2
- 关键变更 3

## 新增与改进

### 功能领域 1

- 变更描述
- 变更描述

### 功能领域 2

- 变更描述
- 变更描述

## 安装 / 升级

从 GitHub Releases 下载：

```bash
# Linux amd64
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/vX.Y.Z/ocr-review-publisher_X.Y.Z_linux_amd64.tar.gz
tar xzf ocr-review-publisher_X.Y.Z_linux_amd64.tar.gz

# macOS arm64
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/vX.Y.Z/ocr-review-publisher_X.Y.Z_darwin_arm64.tar.gz
tar xzf ocr-review-publisher_X.Y.Z_darwin_arm64.tar.gz
```

校验 checksums：

```bash
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/vX.Y.Z/ocr-review-publisher_X.Y.Z_checksums.txt
sha256sum -c ocr-review-publisher_X.Y.Z_checksums.txt
```

## 兼容性

- 已验证 OCR 版本范围：vX.Y.Z - vA.B.C
- 支持平台：GitLab（包括 13.12 自托管）
- 支持操作系统：darwin、linux
- 支持架构：amd64、arm64

## 验证

- `make release-readiness` 通过
- GitLab e2e 测试：通过 / 跳过（原因）
- OCR 兼容性：通过

## 已知限制

- 仅支持 Open Code Review 输出
- 仅支持 GitLab MR 发布（不支持 GitHub PR）
- 不支持 webhook/服务器模式
- 不支持一键平台建议

## 升级说明

- 配置或标记变更说明
- 破坏性变更说明
