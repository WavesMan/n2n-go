# Git 操作指南与 Actions/Release 标准

本文规范本仓库的分支策略、标签与发布流程，并说明 GitHub Actions 的触发条件与产物命名。

## 仓库与工作流位置
- 工作流文件：`n2n-go\.github\workflows\go.yml`
- 触发条件：仅在“推送标签”时触发（普通提交不触发），见 `go/.github/workflows/go.yml:3-6`
  - 支持两类标签：`v*` 与 语义化数字标签（如 `0.1.0`），见 `go/.github/workflows/go.yml:3-6`

## 分支模型
- `dev`：主开发分支，所有常规开发在该分支进行。
- `vX.Y.Z`：发布冻结分支，例如 `v0.1.0`。创建后不再继续开发，仅用于记录发布时的代码快照。
  - 冻结分支名称与标签名称建议区分，以避免同名歧义（例如分支 `v0.1.0`，标签 `0.1.0`）。

## 发布与触发规则
- 触发动作的是“推送标签”，不是分支推送。
- 工作流触发匹配：
  - `v*`（例如 `v0.1.0`）
  - `[0-9]*.[0-9]*.[0-9]*`（例如 `0.1.0`）
- 推荐使用“注释标签”（annotated tag），可携带发布信息：
  - 示例：`git tag -a 0.1.0 -m "release 0.1.0"`

## 标准发布流程（从 dev 准备发布）
1. 在 `dev` 完成开发并通过自测。
2. 创建冻结分支（保留快照）：
   - `git checkout -b v0.1.0`
   - 将该分支推送到远端（可选）：`git push origin v0.1.0`
3. 在冻结分支当前提交上创建发布标签（与分支名区分）：
   - 注释标签：`git tag -a 0.1.0 -m "release 0.1.0"`
4. 推送标签以触发 Actions：
   - `git push origin 0.1.0`
5. 观察 GitHub Actions 运行并在标签对应的 Release 页面出现产物。

> 若使用 `v0.1.0` 作为标签，同样会触发；当前工作流同时支持 `v*` 与纯数字版本标签。

## 二进制产物与命名
- 构建矩阵（示例，详见工作流）：Linux `amd64/386/armv7/arm64`，macOS `amd64/arm64`，Windows `armv7/arm64`。
- 产物命名：
  - `n2n-go-supernode_<GOOS>_<ARCH>[.exe]`
  - `n2n-go-edge_<GOOS>_<ARCH>[.exe]`
- 发布到对应标签的 Release，作为单独文件（非压缩包）。

## 冻结与后续维护
- 冻结分支不再接受新提交；所有后续开发继续在 `dev`。
- 如需修复并发布新版本：
  - 在 `dev` 中完成修复；创建新的冻结分支 `v0.1.1`；打标签 `0.1.1`；推送标签触发发布。

## 回滚与重发
- 删除本地标签：`git tag -d 0.1.0`
- 删除远端标签：`git push origin :refs/tags/0.1.0`
- 重新打标签并推送：`git tag -a 0.1.0 -m "release 0.1.0" && git push origin 0.1.0`

## 常见问题排查
- “提交不触发”
  - 原因：工作流仅监听 `push.tags`；普通 `commit/push` 不触发。
  - 需要在工作流中添加 `push.branches`/`pull_request.branches` 才能在分支提交/PR 上触发。
- “标签推送不触发”
  - 标签命名未匹配：确保为 `vX.Y.Z` 或 `X.Y.Z`。
  - 未推送标签：执行 `git push origin <tag>`。
  - 仓库 Actions 被禁用：在 GitHub Settings → Actions 启用。
- “分支与标签同名冲突”
  - 避免同名。推荐分支 `vX.Y.Z` 与标签 `X.Y.Z` 的区分命名。

## 权限说明
- 工作流中启用了 `permissions: contents: write`，用于在发布时写入 Release 资产。

