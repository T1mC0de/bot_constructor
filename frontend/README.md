# Frontend

Кабинет продавца: редактор анкеты и список заявок. Vite + React + TypeScript.

Нужен Node.js 20+ и запущенный core (`make run-core` в `backend/`): dev-сервер
проксирует `/api` на `http://localhost:8080`, поэтому CORS в core не нужен.

```bash
npm install
npm run dev     # http://localhost:5173
npm test
npm run lint    # tsc + oxlint
npm run build
```
