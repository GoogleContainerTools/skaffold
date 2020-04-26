FROM node:12.16.0-alpine3.10

ARG ENV=production

RUN mkdir /app
WORKDIR /app

ENV NODE_ENV $ENV

CMD npm run $NODE_ENV

COPY package* ./
RUN npm ci

COPY . .

EXPOSE 3000
