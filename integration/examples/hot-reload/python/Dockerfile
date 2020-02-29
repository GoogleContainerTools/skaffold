FROM python:3.7.4-alpine3.10

RUN pip install --upgrade pip

RUN adduser -D python
USER python
WORKDIR /home/python

ARG DEBUG=0
ENV FLASK_DEBUG $DEBUG
ENV FLASK_APP=src/app.py
CMD ["python", "-m", "flask", "run", "--host=0.0.0.0"]

COPY --chown=python:python requirements.txt .
ENV PATH="/home/python/.local/bin:${PATH}"
RUN pip install --user -r requirements.txt

COPY --chown=python:python src src
