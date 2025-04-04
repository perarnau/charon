FROM nvcr.io/nvidia/tensorrt:24.08-py3

RUN apt-get update \
  && apt-get install -y \
  nano \
  wget \
  curl \
  && wget -O /dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.5/dumb-init_1.2.5_x86_64 \
  && chmod +x /dumb-init

ADD ptychonn_512_bsz8_fp16.trt /app/model_512_fp16.trt
ADD ptychonn_512_bsz8_fp32.trt /app/model_512_fp32.trt
ADD https://web.lcrc.anl.gov/public/waggle/models/ptychonn/diff_scan_810.npy /app/scan.npy

WORKDIR /app

COPY requirements.txt /app
RUN pip3 install --upgrade pip \
  && pip3 install -r requirements.txt

COPY inferPtychoNN.py inferPtychoNNImageProcessor.py helper.py /app/
ENV PYTHONPATH="${PYTHONPATH}:/app"

RUN chmod +x $APP_PATH/entry.sh

ENTRYPOINT ["/dumb-init", "--"]
CMD ["sh", "-c", "$APP_PATH/entry.sh"]

# For Prometheus
# EXPOSE 9100