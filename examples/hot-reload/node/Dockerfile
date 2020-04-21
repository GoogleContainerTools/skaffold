FROM node:12.16.0-alpine3.10

USER node
RUN mkdir /home/node/app
WORKDIR /home/node/app

EXPOSE 3000
ARG ENV=production
ENV NODE_ENV $ENV
CMD npm run $NODE_ENV

COPY --chown=node:node package* ./
RUN npm ci
COPY --chown=node:node . .
