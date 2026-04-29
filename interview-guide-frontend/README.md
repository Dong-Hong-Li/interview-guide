# interview-guide（前端）

Vite + React + TypeScript。开发时通过 **环境变量** 配置 API 代理，仓库里**不提交** `.env.development`（每人本机一份）。

## 配置文件在哪

| 文件 | 是否提交仓库 | 说明 |
|------|----------------|------|
| **`.env.development.example`** | 是 | 模板，含 `VITE_DEV_PROXY_TARGET` 等说明 |
| **`.env.development`** | 否（需自建） | 从 example 复制：`cp .env.development.example .env.development` |

Vite 在 `pnpm dev` / `npm run dev` 且 `mode=development` 时会从 **`interview-guide/` 目录** 读取 `.env.development`。没有该文件时，代理目标用 **`vite.config.ts` 里的默认值**（当前为 `http://127.0.0.1:8081`）。

与 Docker / 本机调试 Go 的对应关系见仓库根目录 **`README.Docker.md`**。

## 常用变量

- **`VITE_DEV_PROXY_TARGET`**：`/api` 开发代理转发的后端根地址（默认 `8081` 对齐 compose）。
- **`VITE_API_BASE_URL`**（可选）：见 `src/api/request.ts`；不设则用相对路径走代理。

修改 `.env.development` 后请**重启**开发服务器。

---

以下为创建项目时的 Vite 模板说明（ESLint 扩展等），可按需保留或删除。

Currently, two official plugins are available:

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react) uses [Babel](https://babeljs.io/) or [oxc](https://oxc.rs) with [rolldown-vite](https://vite.dev/guide/rolldown)
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc) uses [SWC](https://swc.rs)

## Expanding the ESLint configuration

若需更严格的 TypeScript ESLint，可参考模板中的 `eslint.config.js` 与 Vite 文档。
