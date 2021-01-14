FROM node:12.16.0-alpine3.10

WORKDIR /app
EXPOSE 8080
CMD ["npm", "run", "dev"]

COPY package* ./
# examples don't use package-lock.json to minimize updates 
RUN npm install --no-package-lock
COPY . .
