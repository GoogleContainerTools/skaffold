FROM node:8.12.0-alpine

ARG SCRIPT
ENV SCRIPT ${SCRIPT}
WORKDIR /opt/backend
EXPOSE 3000
CMD ["npm", "run", ${SCRIPT}]

COPY . .
RUN npm install
