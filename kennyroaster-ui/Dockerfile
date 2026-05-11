# ---- Build stage ----
FROM node:20-alpine AS builder
WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci --no-audit --no-fund

COPY . .
RUN npm run build

# ---- Production stage ----
FROM nginx:alpine

# Remove default config
RUN rm /etc/nginx/conf.d/default.conf

# FIX: correct destination path
COPY nginx.conf /etc/nginx/nginx.conf

# Copy built frontend
COPY --from=builder /app/dist /usr/share/nginx/html

ENV PORT=8080
EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]