FROM python:3.7.4-alpine3.10
CMD ["python", "-m", "flask", "run", "--host=0.0.0.0"]
ENV FLASK_DEBUG=1
ENV FLASK_APP=app.py

COPY requirements.txt .
RUN pip install -r requirements.txt
COPY src ./

