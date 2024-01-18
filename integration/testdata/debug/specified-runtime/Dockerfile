FROM node:14.9-alpine

WORKDIR /opt/backend
EXPOSE 3000
CMD ["node", "src/index.js"]
# add this so skaffold will infer env so that DlvTransformer will be used for go runtime if user not specifies runtime when building image
ENV GOGC off

COPY . .
RUN npm install --no-package-lock
