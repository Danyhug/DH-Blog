# DH-Blog 一体化部署

本目录包含 DH-Blog 的一体化部署文件，将前端和后端打包到一个可执行文件中。

## 使用方法

1. 选择适合您操作系统的可执行文件（均在 `build/` 目录下）：

```
build/dhblog-darwin-arm64      # macOS ARM架构 (M1/M2/M3芯片)
build/dhblog-windows-amd64.exe # Windows 64位
build/dhblog-linux-amd64       # Linux 64位
```

2. 直接运行可执行文件：

```bash
# macOS
./build/dhblog-darwin-arm64

# Linux
./build/dhblog-linux-amd64

# Windows
build\dhblog-windows-amd64.exe
```

3. 首次运行会在同级目录下创建 `data` 目录，用于存储数据库文件、配置文件和上传的文件。

4. 访问地址：http://localhost:2233

## 目录结构

```
blog-deploy/
├── build/
│   ├── dhblog-darwin-arm64      # macOS ARM架构可执行文件
│   ├── dhblog-windows-amd64.exe # Windows 64位可执行文件
│   ├── dhblog-linux-amd64       # Linux 64位可执行文件
├── data/                        # 数据目录（自动创建）
│   ├── config.yaml              # 配置文件
│   ├── dhblog.db                # SQLite数据库文件
│   ├── upload/                  # 上传文件目录
│   └── files/                   # 文件存储目录
├── build.sh                     # 一键构建脚本（Linux/macOS）
├── build.bat                    # 一键构建脚本（Windows）
└── README.md                    # 说明文档
```

## 构建方法

如需自行构建，直接在 `blog-deploy` 目录下执行：

- **Linux/macOS**：

```bash
chmod +x build.sh
./build.sh
```

- **Windows**：

```
build.bat
```

构建完成后，所有平台的可执行文件都在 `build/` 目录下。构建过程中的临时文件会自动清理。
