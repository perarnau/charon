FROM nvcr.io/nvidia/tensorrt:24.08-py3

RUN apt-get update \
  && apt-get install -y \
  nano \
  wget \
  curl \
  && wget -O /dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.5/dumb-init_1.2.5_x86_64 \
  && chmod +x /dumb-init

ENV APP_PATH="/opt/app"

ADD ptychonn_512_bsz8_fp16.trt $APP_PATH/model_512_fp16.trt
ADD ptychonn_512_bsz8_fp32.trt $APP_PATH/model_512_fp32.trt
ADD https://web.lcrc.anl.gov/public/waggle/models/ptychonn/diff_scan_810.npy $APP_PATH/scan.npy

COPY requirements.txt $APP_PATH/
RUN pip3 install --upgrade pip \
  && pip3 install -r $APP_PATH/requirements.txt

COPY pvapy-inference.py helper.py entry.sh $APP_PATH/
ENV PYTHONPATH="${PYTHONPATH}:${APP_PATH}"

RUN chmod +x $APP_PATH/entry.sh

WORKDIR $APP_PATH
ENTRYPOINT ["/dumb-init", "--"]
CMD ["sh", "-c", "$APP_PATH/entry.sh"]

# For Prometheus
# EXPOSE 9100