> English version: [release.md](release.md)

# 发布流程

本项目是 [Open Code Review (OCR)](https://github.com/alibaba/open-code-review) 输出的封装层。发布必须同时证明本地代码正确性和与 OCR CLI 输出契约的兼容性。

## 发布产物

每次发布生成：

- `ocr-review-publisher_<version>_darwin_amd64.tar.gz`
- `ocr-review-publisher_<version>_darwin_arm64.tar.gz`
- `ocr-review-publisher_<version>_linux_amd64.tar.gz`
- `ocr-review-publisher_<version>_linux_arm64.tar.gz`
- `ocr-review-publisher_<version>_checksums.txt`

不支持：Homebrew、npm、Docker。

## 本地门禁

打标签前运行：

```bash
make check
make test-compat
```

这些不需要密钥或网络访问。`make check` 包含格式、测试、vet、构建和兼容性检查。

更严格的发布前验证：

```bash
make release-readiness
```

这运行 `make check` 加竞态检测测试、空白检查和敏感模式扫描。

如果 GitLab 凭据可用，还应运行：

```bash
make test-e2e-gitlab
```

然后对真实测试 MR 运行本地冒烟流程：

```bash
scripts/gitlab-smoke.example.sh publish
scripts/gitlab-smoke.example.sh check
scripts/gitlab-smoke.example.sh cleanup
```

冒烟脚本构建当前发布器，将评论发布到真实 GitLab MR，并断言渲染的评论质量。

## 发布流程

1. **准备双语发布说明：**
   - 创建 `docs/releases/release-notes-vX.Y.Z.md`（英文）
   - 创建 `docs/releases/release-notes-vX.Y.Z.zh-CN.md`（中文）
   - 使用 `docs/releases/release-notes-template*.md` 中的模板

2. **运行本地门禁：**
   ```bash
   make release-readiness
   make test-e2e-gitlab  # 如果 GitLab 凭据可用
   ```

3. **创建并推送注释标签：**
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   git push origin vX.Y.Z
   ```

4. **GitHub Actions 发布工作流自动运行：**
   - 运行 `make release-readiness`
   - 验证发布说明文件存在
   - 运行 GoReleaser 构建和发布产物
   - 创建包含双语说明的 GitHub Release

5. **验证发布：**
   - 检查 GitHub Release 页面确认产物正确
   - 验证校验和文件
   - 运行发布二进制冒烟门禁：
     ```bash
     make smoke-release-binary
     # 或验证特定版本：
     OCR_RELEASE_TAG=v0.1.1 make smoke-release-binary
     ```
   - 该命令下载当前平台对应的归档文件，解压二进制，并验证 `version` 和 `help` 命令输出是否符合预期。

## GitHub Actions 门禁

必需的工作流：

- **CI**（`.github/workflows/ci.yml`）：在推送到 main 和 PR 时运行 `make check`。
- **OCR 兼容性**（`.github/workflows/ocr-compatibility.yml`）：每周运行固定兼容性测试；手动触发时可选运行实时捕获（需配置 LLM 密钥）。
- **发布就绪**（`.github/workflows/release-readiness.yml`）：仅手动触发时运行 `make release-readiness`。不发布，不创建标签/发布。
- **发布**（`.github/workflows/release.yml`）：标签触发，构建产物并发布 GitHub Release。

发布就绪工作流不发布任何内容。它仅验证代码库通过所有发布前门禁。真实 GitLab e2e/冒烟仍是需要平台凭据的手动可选步骤。

## 发布阻止器

以下情况不得发布：

- 解析器兼容性对最新已验证的 OCR 输出失败
- JSON 模式包含破坏机器解析的人类聊天内容
- 渲染的评论未通过质量检查清单
- GitLab clear/update 可以删除未标记的评论
- 重复发布创建重复摘要
- 令牌或本地环境细节出现在已跟踪文件中
- 发布说明未注明已验证的 OCR 版本范围
- README.md 和 README.zh-CN.md 缺失、过时或不一致
- README 徽章未遵循本地 readme-badges 技能

## 发布说明

每次发布需要双语发布说明：

- `docs/releases/release-notes-vX.Y.Z.md`（英文）
- `docs/releases/release-notes-vX.Y.Z.zh-CN.md`（中文）

说明应包含：

- 发布器版本
- 已验证的 OCR 版本范围
- 支持的平台范围
- 已知 OCR 输出兼容性限制
- 是否运行了 GitLab e2e/冒烟
- 安装说明
- 配置或标记的迁移说明
