# n2n-go

n2n-go 是基于 Go 的轻量级 n2n 兼容实现，提供 supernode 与 edge 组件，支持在现代环境中快速部署虚拟二层网络，并与原生 C 版 n2n（3.x）进行数据面互通。

## 快速开始
- 启动 supernode（公开 UDP 端口）：
  - `go run cmd/supernode/main.go -bind 0.0.0.0 -p 7654 -t 5645 -v 1`
  - 打开防火墙 UDP 入站端口（示例：`7654`）。
- 启动 edge（加入社区并注册到 supernode）：
  - `go run cmd/edge/main.go -c mynetwork -k mysecret -l <supernode_ip>:7654 -p 7655 -v 1`
  - 可选设置本地绑定地址：`-bind 0.0.0.0`。

当两个 edge 使用相同社区（`-c`）与同一个 supernode（`-l`）时，即可在虚拟网络中互通。

## 与 C 侧兼容
- 消息头版本与字段编码对齐，支持 C 侧将类型编码在 `flags` 低 5 位的头格式；`register`/`packet` 等核心消息互通已在集成测试与实网验证中通过。
- 数据包头字段顺序与 C 版一致（`transform` 在前，`compression` 在后），并对不一致情况做了解码容错。
- 注册体的可选字段（`auth`/`key_time`）按剩余长度容错解析，避免不同构建的差异导致失败。

## 管理与日志
- 管理端口默认仅监听本机：`supernode -t 5645`、`edge -t 5644`；支持：
  - `w mgmt verbose <n>`：动态调整日志级别（`n=0/1/2`）。
  - `w mgmt stop`：停止（同机调用）。
- 日志级别：
  - `-v 0`：基础输出
  - `-v 1`：事件（注册/查询/转发）
  - `-v 2`：数据面细节（包收发）

## 典型参数
- supernode：
  - `-bind <addr>` 绑定地址（默认 `0.0.0.0`）
  - `-p <port>` 数据端口（默认 `7654`）
  - `-t <port>` 管理端口（默认 `5645`）
  - `-v <level>` 日志级别（默认 `0`）
- edge：
  - `-c <community>` 社区名（默认 `community`，需与对端一致）
  - `-l <sn_ip:port>` supernode 地址（默认 `127.0.0.1:7654`）
  - `-p <port>` 本地 UDP 端口（默认 `7655`）
  - `-bind <addr>` 本地绑定地址（默认 `0.0.0.0`）
  - `-k <key>` 加密密钥（启用后将使用 `-A` 指定的算法）
  - `-A <aes|chacha|null>` 变换算法（默认 `null`；与 C 版 `AES=3`、`ChaCha20=4` 兼容）
  - `-z <none|zstd>` 压缩算法（默认 `none`；与 C 版 `ZSTD=3` 兼容）
  - `-H` 启用安全头模式（在 AEAD 下封装头部，提升安全性）
  - `-t <port>` 管理端口（默认 `5644`）
  - `-v <level>` 日志级别（默认 `0`）

## systemd 集成
- 项目已提供示例 unit 文件：
  - `go/packages/etc/systemd/system/supernode.service`
  - `go/packages/etc/systemd/system/edge.service`
- 使用方法：
  - 将 unit 文件安装至系统（例如 `/etc/systemd/system/`），根据需要调整 `ExecStart` 与参数。
  - `sudo systemctl daemon-reload`
  - `sudo systemctl start supernode` / `sudo systemctl enable supernode`
  - `sudo systemctl start edge` / `sudo systemctl enable edge`

## 实网验证建议
- 公网 supernode：开放 UDP `<数据端口>`（如 `7654`）入站；云安全组与 OS 防火墙同时配置。
- 社区名需一致；密钥/算法组合在两端一致时可进行加密传输（默认 `null` 便于先验证互通，再启用加密）。
- 当转发目的 MAC 未注册时，supernode 会回显（`echo`）；当目标注册后自动转发（`forward`）。

## 构建
- 使用 Go 1.22+：
  - `go run cmd/supernode/main.go ...`
  - `go run cmd/edge/main.go ...`
  - 可使用 `go build` 生成二进制供 systemd 等部署。

## 安全说明
- 提供 AEAD（AES-GCM、ChaCha20-Poly1305）负载加密；在 `-H` 模式下可对头部进行 AEAD 封装，增强元数据保护。
- 不在日志与配置中输出密钥明文；建议使用自定义社区与密钥。

## 兼容性注意
- 管理面协议与 C 侧不同（Go 端为 JSON），数据面已对齐；如需与 C 侧强一致的管理 API，可扩展适配层。
- 旧客户端头格式（类型在 `flags`）可解析；注册体可选字段容错；数据包头字段顺序容错处理已加入。
