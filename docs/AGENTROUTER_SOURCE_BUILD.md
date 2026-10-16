# 含 AgentRouter 修复的 Windows 源码构建

本次修改基于 Cockpit Tools v1.3.70，基线提交为
`7fa0a092df0f1f1070019782e8608e441a6dda09`，本地分支为
`codex/fix-agentrouter-client-headers`。

## 修复内容

Provider Gateway 新建上游请求时，会丢失 Codex 请求原有的 `User-Agent` 和
`Originator`。AgentRouter 因此可能返回 `401 unauthorized client detected`。

`sidecars/cockpit-cliproxy/provider_gateway.go` 现在仅在上游主机名为
`agentrouter.org` 时透传这两个真实请求头。缺少的标识不会补造，其他供应商的
行为不变；上游 `Authorization` 仍由供应商 API Key 生成，不复制客户端的
Authorization、Cookie 或 ChatGPT 账号头。

修复覆盖 Responses 和 Chat Completions 两种 Provider Gateway 协议。
它不包含当前电脑的账号、API Key 或混合模型路由配置，也不修改官方 OAuth 执行路径。

## 本地构建的推广默认设置

根目录 `build-config.json` 中的 `promotionsEnabled` 默认为 `false`，前端与 Rust
后端共用这一配置。该构建关闭顶部广告、赞助商推荐入口、赞助商线路自动同步及
关于页面的赞赏入口；远端数据无法覆盖这个本地开关，已有广告缓存也不会在首屏显示。
`announcements.json` 内的推广开关和用户设置中的顶部推广默认值也已关闭。
普通功能公告、项目作者信息和用户自行配置的供应商继续保留。

`automaticUpdatesEnabled` 同样默认为 `false`：禁用启动及每小时更新检查、更新提醒、
静默下载和升级说明自动弹窗。后端会屏蔽旧安装中保存的自动检查、自动安装和提醒设置，
设置界面的相关自动更新开关也会禁用。手动检查更新入口继续保留。

## 构建环境

适用于 Windows x64。先安装以下依赖，再打开新的终端：

- Git。
- Node.js 22 LTS，至少 22.12，附带 npm。
- Rust stable，使用 `x86_64-pc-windows-msvc` 工具链，确保 `cargo` 可用。
- Visual Studio 2022 或 Build Tools，安装“使用 C++ 的桌面开发”、MSVC v143 和 Windows SDK。
- Go 1.26 或更新版本，确保 `go` 可用；最低版本以侧车的 `go.mod` 为准。
- Microsoft Edge WebView2 Runtime，用于运行桌面应用。

首次构建需要联网获取 npm、Cargo、Go 依赖及 Tauri 安装包工具。
Visual Studio 安装在自定义目录时，推荐使用它的 **Developer PowerShell for VS 2022**。
仓库的构建脚本仅自动识别几种默认安装路径。

## 生成安装包

在这份已修改源码的仓库根目录执行。换电脑构建时，应复制这份源码及本地改动；
重新克隆上游仓库不会包含尚未提交到上游的修复。

```powershell
node --version
npm --version
cargo --version
go version

npm ci
if ($LASTEXITCODE -ne 0) { throw 'npm ci 失败' }

# 确保本次从修改后的 Go 源码重新构建网关。
Remove-Item Env:COCKPIT_SKIP_CLIPROXY_BUILD -ErrorAction SilentlyContinue

# 使用仓库已有的 CI 配置，关闭官方更新签名产物的生成。
npm run tauri -- build --bundles nsis --config src-tauri/tauri.ci.conf.json
if ($LASTEXITCODE -ne 0) { throw 'Cockpit 安装包构建失败' }
```

此配置只关闭更新签名产物的生成，无需官方更新私钥，应用版本仍为 1.3.70。
构建走 Tauri CLI，会先打包前端，再由 `src-tauri/build.rs` 编译本地 Go 侧车，
最后将其加入安装包。完整安装包不应通过直接执行 `cargo build` 生成。

默认产物位置（未设置 `CARGO_TARGET_DIR`、未额外指定 `--target`）：

```text
target/release/bundle/nsis/*-setup.exe
sidecars/cockpit-cliproxy/bin/cockpit-cliproxy-x86_64-pc-windows-msvc.exe
```

把生成的 **NSIS 安装程序** 复制到其他 Windows x64 电脑安装即可，
无需再手动替换网关文件。安装前完全退出 Cockpit，包括托盘中的实例。
其他电脑仍需在 Cockpit 中配置各自的官方账号、AgentRouter API Key 和混合模型路由，
再通过 Cockpit 重启对应 Codex 实例。

该安装包保留上游的手动更新来源，但默认不自动检查、推送或下载更新。
以后主动安装官方版本会替换本地修改；上游尚未合入修复时，
升级后需要重新将修复应用到新源码并构建。

## 定向验证

请求头回归测试使用内存中的模拟上游，不访问 AgentRouter，也不需要真实密钥：

```powershell
Push-Location sidecars/cockpit-cliproxy
try {
    go test . -run '^TestProviderGatewayClientHeaders' -count=1
    if ($LASTEXITCODE -ne 0) { throw '客户端头回归测试失败' }
} finally {
    Pop-Location
}
```

测试检查两个上游协议、域名大小写及端口、缺失标识、其他域名和相似域名，
并检查供应商 Authorization、客户端凭据隔离及原有诊断头。

安装后分别在 Codex 菜单选择官方 GPT 与 `agentrouter / deepseek-v4-flash`，
各发送一条消息验证实际账号和路由配置。离线回归测试不能代替目标电脑上的这一步。
