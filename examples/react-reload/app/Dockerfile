FROM node:14.9-alpine

WORKDIR /app
EXPOSE 8080
CMD ["npm", "run", "dev"]

COPY package* ./
# examples don't use package-lock.json to minimize updates 
RUN npm install --no-package-lock
COPY . .
