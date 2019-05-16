FROM node:8.12.0-alpine

WORKDIR /opt/backend
EXPOSE 3000
CMD ["node", "src/index.js"]

COPY . .
RUN npm install
