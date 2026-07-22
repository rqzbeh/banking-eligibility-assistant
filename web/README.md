# رابط کاربری — دستیار هوشمند بانکی

React + TypeScript. خروجی build در `dist/` توسط gateway (`agent/server.py`) سرو می‌شود.

## توسعه

```bash
npm ci
npm run dev
```

پروکسی dev به backend `:8080` و agent `:8501` وصل است.

## ساخت

```bash
npm ci
npm run build
# → dist/
```

در Docker، stage مربوط به web همین build را انجام می‌دهد.
