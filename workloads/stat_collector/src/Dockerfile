FROM python:3.10

WORKDIR /app
COPY . /app
RUN pip3 install --upgrade pip \
  && pip3 install -r requirements.txt

EXPOSE 8501
ENV PYTHONPATH "${PYTHONPATH}:/app"