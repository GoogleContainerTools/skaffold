FROM node:14.9-alpine

WORKDIR /opt/backend
EXPOSE 3000
CMD ["npm", "start"]

COPY . .
RUN npm install --no-package-lock
