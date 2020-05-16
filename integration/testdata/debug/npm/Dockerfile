FROM node:12.16.0-alpine3.10

WORKDIR /opt/backend
EXPOSE 3000
CMD ["npm", "start"]

COPY . .
RUN npm ci
