FROM python:3.9-alpine

RUN pip install --upgrade pip

RUN adduser -D python

ARG DEBUG=0
ENV FLASK_DEBUG $DEBUG
ENV FLASK_APP=src/app.py
CMD ["python", "-m", "flask", "run", "--host=0.0.0.0"]

COPY --chown=python:python requirements.txt .
RUN pip install -r requirements.txt

COPY --chown=python:python src src
