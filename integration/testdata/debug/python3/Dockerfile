FROM python:3.9-alpine

EXPOSE 5000
# Use shell-style CMD so that the image uses ["/bin/sh", "-c", "npm start"]
CMD python3 app.py

WORKDIR /app
COPY . /app
RUN pip3 install -r requirements.txt
