# Web App (Hub MVP)

This app renders a static-first skills catalog from `../../registry/index.json`.

## Local development

```bash
cd apps/web
npm install
npm run dev
```

Then open `http://localhost:3000`.

## Build

```bash
npm run build
npm start
```

## Routes

- `/` overview
- `/skills` catalog with filters
- `/skills/<category>/<slug>` skill details with install snippets
