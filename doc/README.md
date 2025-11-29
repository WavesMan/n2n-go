# n2n-go 使用说明

## 组件
- `edge`：连接 TAP 设备，封装以太帧为 n2n 报文，与超级节点交互。
- `supernode`：维护 peers，处理注册、查询与数据转发。

## 管理 API
- 请求：`<type> <tag[:flags[:auth]]> <method> [params]`
- 回复：`begin/row/end/error/subscribed` 行式 JSON，详见 `n2n/doc/ManagementAPI.md`。

## 安全
- AEAD 使用随机唯一 nonce，密钥通过 HKDF(sha256) 派生 32 字节。
- `-A aes|chacha|null`，`-z none|zstd` 与实际处理一致性校验。

## 运行示例
```sh
supernode -p 7654 -t 5645 -bind 0.0.0.0
edge -c community -k mysecret -l 127.0.0.1:7654 -p 7655
```

## systemd
- 参考 `go/packages/etc/systemd/system/*.service` 配置服务。
