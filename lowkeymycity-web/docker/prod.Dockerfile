FROM node:22-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .

ARG VITE_USE_MOCKS
ARG VITE_API_URL
ARG VITE_APP_URL
ARG VITE_POSTHOG_KEY
ARG VITE_POSTHOG_HOST
ENV VITE_USE_MOCKS=$VITE_USE_MOCKS \
    VITE_API_URL=$VITE_API_URL \
    VITE_APP_URL=$VITE_APP_URL \
    VITE_POSTHOG_KEY=$VITE_POSTHOG_KEY \
    VITE_POSTHOG_HOST=$VITE_POSTHOG_HOST

RUN npm run build


FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
# the image's entrypoint envsubsts ${API_PORT} from the container env
# into /etc/nginx/conf.d/default.conf at startup
COPY docker/nginx.conf.template /etc/nginx/templates/default.conf.template

EXPOSE 80
