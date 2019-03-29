FROM node:alpine

WORKDIR /app
COPY package.json ./
RUN ["npm", "install"]

FROM node:alpine

WORKDIR /app

COPY --from=0 /app/node_modules node_modules

EXPOSE 8080
CMD ["npm", "run", "dev"]

COPY . .
