FROM node:8.12.0-alpine

WORKDIR /app
EXPOSE 8080
CMD ["npm", "run", "dev"]

COPY package* ./
RUN npm install
COPY . .
