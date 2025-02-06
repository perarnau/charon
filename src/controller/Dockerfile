# The base image should have the kuberenetes and nrm libraries installed
FROM gemblerz/nrm:0.1.0

COPY requirements.txt /app/
RUN pip3 install -r /app/requirements.txt

COPY controller.py /app/
ENTRYPOINT ["python3", "/app/controller.py"]