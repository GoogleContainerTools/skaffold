FROM python:3.7.3-alpine3.9
CMD ["python", "-m", "flask", "run", "--host=0.0.0.0"]
ENV FLASK_DEBUG=1
ENV FLASK_APP=app.py

COPY requirements.txt .
RUN pip install -r requirements.txt
COPY src ./

