# ocr-review-publisher v0.1.3 发布说明

发布日期：2026-06-10

## 概览

v0.1.2 之后的补丁版本。GitLab CI 冒烟测试在 OCR npm 包下载不稳定时增加了重试机制。解析器、渲染器和发布器的行为没有改动。

## 变更内容

### GitLab CI 冒烟测试

- OCR 二进制的 `npm install` 增加 3 次重试、每次间隔 15 秒——Docker runner 容器内从 github.com 下载经常超时
- 默认 `PUBLISHER_VERSION` 从 `v0.1.1` 更新为 `v0.1.2`

## 兼容性

- 已验证 OCR 版本：v1.1.13
- 支持平台：GitLab（包括 13.12 自托管）
- 支持操作系统：darwin、linux
- 支持架构：amd64、arm64

## 验证

- `make release-readiness` 通过
- `make smoke-release-binary` 通过
- `make smoke-gitlab-ci` 通过

## 已知限制

- 仅支持 GitLab：不支持 GitHub PR 发布
- 无 webhook、服务器或斜杠命令模式
- 无 Homebrew、npm 或 Docker 包
- 仅当 OCR 输出包含类别和严重性字段时才渲染对应徽章
- 未启用真正的一键 GitLab 建议

## 升级说明

- 与 v0.1.2 无破坏性变更
- 解析器、渲染器和发布器行为无变动
- 无标记格式变更
