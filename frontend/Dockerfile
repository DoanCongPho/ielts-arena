# ---- build stage ----
FROM node:22-alpine AS builder
WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
# ARG only exists during the build; ENV re-exposes it as an actual env var
# so `vite build` (the RUN below) can read it and bake it into the JS via
# import.meta.env.VITE_API_BASE. Override at build time with:
#   docker build --build-arg VITE_API_BASE=http://192.168.1.50:8080 .
ARG VITE_API_BASE=http://localhost:8080
ENV VITE_API_BASE=$VITE_API_BASE
RUN npm run build

# ---- run stage ----
FROM nginx:1.27-alpine

COPY --from=builder /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80