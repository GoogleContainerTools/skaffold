FROM python:3.7-slim
CMD ["python", "-m", "flask", "run"]
ENV FLASK_DEBUG=1
ENV FLASK_APP=app.py

COPY requirements.txt .
RUN pip install -r requirements.txt
COPY src ./

