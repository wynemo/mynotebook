# MyNotebook

用 [egoist/mygo](https://github.com/egoist/mygo) Native UI 编写的轻量 Markdown 桌面软件，支持 macOS / Windows。界面由 Go 原生组件绘制，不使用 HTML、JavaScript 或 WebView。

## 功能

- 同时打开多个本地文件夹，文件夹和子目录均可折叠，整个侧栏可隐藏、拖动调整宽度。
- 多标签笔记，创建、打开、保存、另存为、重新载入 Markdown 文件。
- 原生文本编辑，支持中文输入法、选择、撤销/重做和标准剪贴板操作。
- 原生 Markdown 预览，支持标题、列表、任务列表、表格、代码块及本地图片；可切换编辑、预览、分屏。
- 粘贴剪贴板图片：先保存笔记，然后使用 Cmd/Ctrl+V。图片保存在笔记同目录的 `.assets/` 下，自动插入相对路径，同一图片自动去重。普通文本照常粘贴。
- 全局搜索所有工作区文件夹内已保存的 Markdown 内容，点击结果跳到对应行。后台搜索支持取消，最多显示 500 条结果。
- 保存前检查外部修改，遇到冲突保留编辑内容；关闭未保存的笔记或退出应用时提供保存提示。
- 记住工作区文件夹、最后打开的文件和侧栏宽度；跟随系统浅色/深色主题。

## 运行

需要 Go 1.27.3 或更新版本。Go 开启自动工具链下载时会按 `go.mod` 获取对应版本。原生项目不需要 Node.js、npm 或 Bun。

```sh
go run .
```

也可以使用 MyGo 开发模式，修改 Go 代码后自动重启：

```sh
go tool mygo dev
```

## 构建

```sh
# 当前系统的应用及安装包
go tool mygo build

# macOS Apple Silicon 和 Windows x64
go tool mygo build -platform darwin/arm64,windows/amd64

# macOS 通用版（Apple Silicon + Intel）
go tool mygo build -platform darwin/universal
```

输出在 `build/`。macOS 输出 `.app` 和 `.dmg`；Windows 输出 `.exe`，安装 NSIS（`makensis`）后还可生成安装程序。发布给其他用户前需自行配置代码签名与 macOS 公证。Windows 版本使用 Native UI，不依赖 WebView2。

## 快捷键

| 操作 | macOS | Windows |
| --- | --- | --- |
| 新建笔记 | Cmd+N | Ctrl+N |
| 打开文件 | Cmd+O | Ctrl+O |
| 添加文件夹 | Cmd+Shift+O | Ctrl+Shift+O |
| 保存 | Cmd+S | Ctrl+S |
| 另存为 | Cmd+Shift+S | Ctrl+Shift+S |
| 关闭笔记 | Cmd+W | Ctrl+W |
| 隐藏/显示侧栏 | Cmd+B | Ctrl+B |
| 全局搜索 | Cmd+Shift+F | Ctrl+Shift+F |
| 切换编辑/预览/分屏 | Cmd+E | Ctrl+E |

## 验证

```sh
go test ./...
go test -race ./...
go vet ./...

# 可选：离屏渲染原生界面的浅色/深色截图
MYNOTEBOOK_SCREENSHOTS=/tmp/mynotebook-screenshots go test -run TestNativeScreenshots -v .
```

测试覆盖文件读写冲突、Unicode 搜索、搜索取消与结果上限、图片附件去重、符号链接边界、工作区恢复，以及真实原生组件的输入、预览、保存提示、标签、侧栏和图片插入撤销。

## 当前边界

文件使用 UTF-8 编码，单篇文档上限 8 MiB。全局搜索只检索 `.md`、`.markdown`、`.mdown`，跳过隐藏目录、符号链接和 `node_modules`。外部新增文件可点击侧栏“刷新”；外部修改的文件可使用“重新载入”。未保存的标签只在当前会话中保留，退出时需保存或明确放弃。

macOS 构建、原生界面离屏渲染及自动测试已验证；Windows 已完成交叉编译，尚未进行 Windows 桌面实测。
