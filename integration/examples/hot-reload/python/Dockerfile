FROM python:3.7.4-alpine3.10
CMD ["python", "-m", "flask", "run", "--host=0.0.0.0"]
ENV FLASK_DEBUG=1
ENV FLASK_APP=app.py
WORKDIR /usr/src/app
COPY requirements.txt .
RUN chown -R 400:400 /usr/src/app && pip install -r requirements.txt
COPY src ./
