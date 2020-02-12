FROM node:10.15.3-alpine

WORKDIR /app
EXPOSE 3000
CMD ["npm", "run", "dev"]

COPY package* ./
RUN chown -R 400:400 /app && npm ci
COPY src .
