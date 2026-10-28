FROM node:24-bookworm-slim AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --legacy-peer-deps
COPY . .
ENV COMMO_RUNTIME=vps NEXT_PUBLIC_RUNTIME=vps NEXT_PUBLIC_SITE_URL=https://pishare.cloud NEXT_TELEMETRY_DISABLED=1
RUN npx next build --webpack
FROM node:24-bookworm-slim AS zalo-worker
WORKDIR /app
COPY deploy/zalo/package*.json ./
RUN npm ci --omit=dev --ignore-scripts
COPY --chown=node:node scripts/zalo-worker.ts ./scripts/zalo-worker.ts
COPY --chown=node:node lib/shop/notifications.ts ./lib/shop/notifications.ts
COPY --chown=node:node lib/payments/sqlite.ts lib/payments/outbox.ts ./lib/payments/
ENV DATA_DIR=/data
USER node
CMD ["node","scripts/zalo-worker.ts"]

FROM node:24-bookworm-slim AS web
WORKDIR /app
ENV NODE_ENV=production COMMO_RUNTIME=vps NEXT_PUBLIC_SITE_URL=https://pishare.cloud DATA_DIR=/data HOSTNAME=0.0.0.0 PORT=3000
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static
COPY --from=build --chown=node:node /app/public ./public
COPY --from=build --chown=node:node /app/scripts/maintenance.ts ./scripts/maintenance.ts
COPY --from=build --chown=node:node /app/lib/shop ./lib/shop
USER node
EXPOSE 3000
CMD ["node","server.js"]
