FROM node:10.15.3-alpine

WORKDIR /app
EXPOSE 8080
CMD ["npm", "run", "dev"]

COPY package* ./
RUN npm ci
COPY . .
