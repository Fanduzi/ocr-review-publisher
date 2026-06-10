# ocr-review-publisher v0.1.2 发布说明

## 概览

v0.1.1 之后的补丁版本。改进了 GitLab CI 文档和发布二进制的冒烟验证。解析器、渲染器和发布器的行为没有改动。

## 变更内容

### GitLab CI 示例

- `docs/ci.md` 中的"基本示例"改为从 GitHub Releases 下载发布二进制（固定版本号、校验 checksum、安装到 `/usr/local/bin`）
- 源码构建作为开发者和自定义 fork 的备选方案保留

### 发布二进制冒烟门禁

- 新增 `scripts/smoke-release-binary.sh`：从 GitHub Releases 下载平台对应的压缩包，解压后验证 `--version` 和 `--help` 输出
- 新增 `make smoke-release-binary` 目标
- 支持通过 `OCR_RELEASE_TAG` 指定测试的发布版本，`OCR_RELEASE_SMOKE_DIR` 指定解压目录

### GitLab CI 冒烟测试（真实发布验证）

- `scripts/smoke-gitlab-ci.sh` 改为从 GitHub Releases 下载发布包和 checksums 文件，不再从源码构建
- 验证完整流程：下载、checksum 校验、二进制执行、发布、清理
- 修复 checksum 校验：保存 tarball 时使用与 `checksums.txt` 条目一致的原始文件名
- 新增可配置代理支持（`OCR_CI_SMOKE_HTTPS_PROXY`）和本地缓存模式（`OCR_CI_SMOKE_LOCAL_CACHE`）

### 文档

- 更新 README.md 和 README.zh-CN.md，补充从 GitHub Releases 下载的说明（含 checksum 校验步骤）
- 更新 `docs/release.md` 和 `docs/release.zh-CN.md`：在"验证发布"步骤中记录发布二进制冒烟门禁
- 更新 `docs/quality-gates.md` 和 `docs/quality-gates.zh-CN.md`：新增发布后门禁，要求 `smoke-release-binary` 通过

## 安装

从 GitHub Releases 下载：

```bash
# Linux amd64
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.2/ocr-review-publisher_0.1.2_linux_amd64.tar.gz
tar xzf ocr-review-publisher_0.1.2_linux_amd64.tar.gz

# macOS arm64（Apple Silicon）
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.2/ocr-review-publisher_0.1.2_darwin_arm64.tar.gz
tar xzf ocr-review-publisher_0.1.2_darwin_arm64.tar.gz
```

校验 checksums：

```bash
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.2/ocr-review-publisher_0.1.2_checksums.txt
sha256sum -c ocr-review-publisher_0.1.2_checksums.txt
```

## 兼容性

- 已验证 OCR 版本：v1.1.13
- 支持平台：GitLab（包括 13.12 自托管）
- 支持操作系统：darwin、linux
- 支持架构：amd64、arm64

## 验证

- `make release-readiness` 通过
- `make smoke-release-binary` 通过
- GitLab CI 冒烟测试：发布二进制下载 + checksum 校验 + 发布/清理流程通过

## 已知限制

- 仅支持 GitLab：不支持 GitHub PR 发布
- 无 webhook、服务器或斜杠命令模式
- 无 Homebrew、npm 或 Docker 包
- 仅当 OCR 输出包含类别和严重性字段时才渲染对应徽章
- 未启用真正的一键 GitLab 建议

## 升级说明

- 与 v0.1.1 无破坏性变更
- 解析器、渲染器和发布器行为无变动
- 无标记格式变更
