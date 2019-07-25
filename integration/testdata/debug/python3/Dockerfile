FROM python:3.7.3-alpine3.9

EXPOSE 5000
CMD ["python3", "app.py"]

WORKDIR /app
COPY . /app
RUN pip3 install -r requirements.txt
