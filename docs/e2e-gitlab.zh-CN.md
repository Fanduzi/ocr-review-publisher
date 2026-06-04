> English version: [e2e-gitlab.md](e2e-gitlab.md)

# GitLab E2E 测试

本项目包含可选的端到端测试，针对真实 GitLab 实例运行。

## 必需环境变量

```bash
export OCR_E2E_GITLAB=1
export OCR_E2E_GITLAB_URL=https://gitlab.example.com
export OCR_E2E_GITLAB_TOKEN=your-token
export OCR_E2E_GITLAB_PROJECT_ID=123
export OCR_E2E_GITLAB_MR_IID=456
```

当 `OCR_E2E_GITLAB=1` 时，五个变量都是必需的。

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

此文件被 git 忽略（`.gitignore` 中的 `env*.local`），不应提交。它包含本地密钥。

当 `env.gitlab.local` 存在时，`make test-e2e-gitlab` 在运行测试前自动 source 它。

## 运行 E2E 测试

```bash
# 使用环境文件（如果存在会自动 source）
make test-e2e-gitlab

# 使用显式环境变量
OCR_E2E_GITLAB=1 go test -tags=e2e ./internal/e2e/gitlab -count=1 -v
```

当 `OCR_E2E_GITLAB` 未设为 `1` 且 `env.gitlab.local` 不存在时，所有 e2e 测试会干净地跳过。

## 测试覆盖内容

1. **创建和清除摘要** - 发布摘要笔记，验证存在，清除，验证已删除
2. **创建和清除内联** - 在新增行上发布内联评论，验证存在，清除，验证已删除
3. **未标记笔记保留** - 创建未标记笔记，运行 clear 操作，验证未标记笔记存活
4. **摘要更新（无重复）** - 发布两次，验证恰好存在一个摘要标记笔记
5. **内联渲染质量** - 发布含所有功能的发现（已有代码、建议、思考、类别、严重性），清理前验证渲染的 Markdown 质量
6. **跳过的内联在摘要诊断中** - 发布含无效行的发现，清理前验证它出现在摘要诊断中

## 评论质量断言

E2E 测试在**清理前**通过获取 GitLab 笔记验证渲染的评论质量：

- 内联标记存在
- 类别和严重性徽章已渲染
- 已有代码在语言感知围栏块中（如 `.go` 文件使用 ` ```go `）
- 建议变更在语言围栏中，而非 ` ```suggestion `
- 审查上下文在 `<details>` 块中
- 思考/审查者笔记存在
- 无原始 GitLab API 错误（`line_code can't be blank`）
- 摘要诊断在 `<details>` 块中，措辞正确

## 安全行为

- 测试使用 `t.Cleanup` 自行清理
- 测试在每次运行前后清除发布器拥有的笔记
- 测试不删除未标记的用户或机器人评论
- 测试可在同一 MR 上安全重跑

## 冒烟脚本

本地冒烟脚本可用于手动验证：

```bash
# 复制示例
cp scripts/gitlab-smoke.example.sh .local/gitlab-smoke.sh

# 编辑 .local/gitlab-smoke.sh 配置本地设置
# 然后运行：
chmod +x .local/gitlab-smoke.sh
.local/gitlab-smoke.sh publish
.local/gitlab-smoke.sh check
.local/gitlab-smoke.sh cleanup
```

## 警告

除非了解影响，否则不要对生产 MR 运行 e2e 测试。测试会在指定 MR 上创建和删除评论。
