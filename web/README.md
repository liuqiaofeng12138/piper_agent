# Piper Agent Web（Phase W2）

Vue 3 + TypeScript 对话界面，风格参考 DeepSeek 网页版。

## 开发

1. 一键后端（Runtime + Worker + 网关）：

   ```powershell
   cd gateway
   go run ./cmd/piper-serve -f ../deploy/config/local.yaml
   ```

2. 安装依赖并启动前端：

   ```powershell
   cd web
   npm install
   npm run dev
   ```

3. 浏览器打开 Vite 提示的地址（默认 `http://127.0.0.1:5173`）。`/api` 由 Vite 代理到 `http://127.0.0.1:8080`。

## 环境变量

- `VITE_API_BASE`：默认 `/api/v1`
