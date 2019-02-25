FROM node:8
WORKDIR /usr/src/app
COPY ./app/* ./
RUN npm install
EXPOSE 80
CMD [ "npm", "start" ]