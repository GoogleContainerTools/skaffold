ARG BASE_PREFIX=mirror.gcr.io/library/
FROM ${BASE_PREFIX}node:14.9-alpine

WORKDIR /opt/backend
EXPOSE 3000
CMD ["node", "src/index.js"]

COPY . .
RUN npm install --no-package-lock
