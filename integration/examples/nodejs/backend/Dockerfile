FROM node:8.12.0-alpine

WORKDIR /opt/backend
EXPOSE 3000
CMD ["npm", "run", "dev"]

COPY . .
RUN npm install
