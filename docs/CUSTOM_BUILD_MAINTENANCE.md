# 定制版的后续更新与备份

当前定制基于上游 v1.3.70（`7fa0a092df0f1f1070019782e8608e441a6dda09`），
维护分支为 `codex/fix-agentrouter-client-headers`。
`origin` 指向 `https://github.com/jlcodes99/cockpit-tools.git`。
`fork` 指向自己的 `https://github.com/kingslou/cockpit-tools.git`，
定制分支保存在该 Fork 中；上游 `main` 用于对照和同步。

## 必须保留的三项定制

1. AgentRouter 请求转发保留真实客户端的 `User-Agent` 和 `Originator`，仅对
   `agentrouter.org` 生效，同时保持供应商 API Key 与客户端凭据隔离。
2. `build-config.json` 的 `promotionsEnabled` 为 `false`，前后端屏蔽推广、
   赞助商线路同步以及旧广告缓存。
3. `automaticUpdatesEnabled` 为 `false`，关闭自动检查、推送、下载和升级说明弹窗，
   同时屏蔽旧安装残留的自动更新偏好。

具体实现和构建命令见 [源码构建说明](AGENTROUTER_SOURCE_BUILD.md)。

## 每次升级的流程

在源码仓库根目录执行。开始前先用 `git status` 检查工作区，
如有新修改，先审查并提交，不要带着未保存的修改合并。

```powershell
git switch codex/fix-agentrouter-client-headers
git status --short

# 每次使用新的备份分支名；保留合并前的提交，便于回退。
git branch backup/custom-before-upgrade-20261016

# 当前仓库最初为浅克隆；首次升级时补齐上游历史。
git rev-parse --is-shallow-repository
# 仅当上一条输出 true 时执行：
git fetch --unshallow origin

git fetch origin --tags
git log --oneline HEAD..origin/main
```

建议先确定要升级的正式发布版本，再合并对应 tag。使用 `git tag --list` 查找
实际存在的 tag 名，不要假定版本标签的格式。下面两种目标选择一种即可：

```powershell
# 跟随上游 main：
git merge origin/main

# 或合并某个正式版本（先将占位文本替换为实际标签）：
# git merge <实际正式版本标签>
```

Git 会保留本地提交，并尝试合并上游改动；同一段代码同时改变时会提示冲突。
发生冲突后，先运行 `git status`，逐个文件结合新实现保留定制行为，
审查后通过 `git add <已解决的文件>` 和 `git merge --continue` 完成。
不要对全部文件直接使用 `ours` 或 `theirs`，否则可能丢失上游修复或定制功能。
如果要放弃尚未完成的本次合并，可以使用 `git merge --abort`。

即使 Git 没报冲突，也要检查三项定制的调用位置和运行效果。
上游若已修复 AgentRouter 问题，应复用其实现并保留行为验证，避免重复处理。
上游新增推广入口、更新任务或重构配置时，也需检查新的代码路径是否遵守本地开关。

按源码构建说明执行定向请求头验证、前端及安装包构建，
再验证旧广告缓存和更新偏好不会重新打开相关功能。
安装后分别用官方 GPT 和 DeepSeek 发送消息，确认实际账号与路由正常。
验证通过后，为新版本建立独立 tag、备份，并保存新的安装包。
未来升级的依赖和构建要求以对应上游版本为准。

完成本地提交后，将定制分支保存到自己的 GitHub 仓库：

```powershell
git push fork codex/fix-agentrouter-client-headers
# 创建新的定制版本标签后，单独推送该标签：
# git push fork <实际定制版本标签>
```

不要用 Fork 页面中的同步操作直接替换定制分支；应在本地审查合并结果后再推送。

## 安装与回退

源码合并后必须重新构建定制安装包，才能在其他电脑使用新版本。
直接安装官方更新包会替换定制程序；当前保留的手动检查更新入口也可能引向官方包，
因此升级时应安装自己重新构建的包。

合并前的备份分支可用于建立单独目录复查或构建旧源码：

```powershell
git worktree add ../cockpit-tools-before-upgrade backup/custom-before-upgrade-20261016
```

程序回退优先使用之前保存的安装包。程序数据格式可能随上游版本改变，
安装新版本前应另外备份应用数据；源码备份不能替代应用数据备份。

## 跨电脑保存源码

已将定制分支和当前定制基线标签推送至自己的 GitHub Fork。
其他电脑可直接克隆定制分支，并添加上游来源：

```powershell
git clone -o fork -b codex/fix-agentrouter-client-headers https://github.com/kingslou/cockpit-tools.git
Set-Location cockpit-tools
git remote add origin https://github.com/jlcodes99/cockpit-tools.git
```

本次另存的 `cockpit-tools-custom-v1.3.70-20261016-repository.zip` 包含已提交源码和
恢复所需的 `.git` 对象、分支、标签及浅克隆边界；解压即可继续维护。
解压时保留隐藏的 `.git` 目录，并用 `git status` 确认状态。
文件名带 `source.zip` 的备份仅含源码，适合阅读或构建，不包含提交历史。
原仓库为浅克隆，直接生成的 bundle 即使通过 `git bundle verify`，也可能无法独立克隆；
因此当前备份使用仓库 ZIP，补齐历史后再使用下面的完整 bundle 流程。

可以保存仓库、源码快照或 Git bundle；完成历史补齐后，推荐创建完整 bundle：

```powershell
git bundle create ../cockpit-tools-custom.bundle --all
git bundle verify ../cockpit-tools-custom.bundle
```

完整 bundle 在其他电脑可以这样恢复：

```powershell
git clone -b codex/fix-agentrouter-client-headers ./cockpit-tools-custom.bundle cockpit-tools
Set-Location cockpit-tools
git remote set-url origin https://github.com/jlcodes99/cockpit-tools.git
```

源码备份不包含安装包、编译工具链、依赖缓存、账号、API Key 或运行时路由配置。
账号和路由需在目标电脑单独配置。不要把工作区的 `.local-state` 目录随源码分发，
该目录可能含有本机敏感状态。
