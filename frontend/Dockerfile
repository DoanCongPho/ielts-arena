# ---- build stage ----
FROM node:22-alpine AS builder
WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
# Empty by default and that is the intended setting: the app calls its own
# origin and nginx.conf proxies /api to the backend, so this image is not
# tied to any environment's API hostname and can be promoted unchanged from
# dev to prod. Only set it to point a build at an API on another origin.
ARG VITE_API_BASE=
ENV VITE_API_BASE=$VITE_API_BASE
RUN npm run build

# ---- run stage ----
FROM nginx:1.27-alpine

COPY --from=builder /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80