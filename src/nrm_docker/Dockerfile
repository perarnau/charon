FROM ubuntu:22.04

RUN apt-get update \
  && apt-get install -y \
  git \
  software-properties-common \
  gcc \
  nano \
  make \
  autoconf \
  automake \
  libtool \
  pkg-config \
  libzmq3-dev \
  libzmq5 \
  libczmq-dev \
  libczmq4 \
  libprotobuf-c-dev \
  protobuf-c-compiler \
  libjansson-dev \
  libjansson4 \
  check \
  libhwloc-dev \
  libpapi-dev \
  mpich \
  libomp-dev \
  libomp5 \
  python3-pip

RUN add-apt-repository ppa:geopm/release \
  && apt-get update \
  && apt-get install -y \
  geopm-service \
  libgeopmd-dev \
  libgeopmd2 \
  python3-geopmdpy

WORKDIR /app
RUN cd /app \
  && git clone https://github.com/bats-core/bats-core.git \
  && cd /app/bats-core \
  && ./install.sh /usr

RUN cd /app \
  && git clone https://github.com/anlsys/libnrm.git \
  && cd /app/libnrm \
  && git checkout feature/prometheus-exporter \
  && ./autogen.sh \
  && ./configure --prefix=/usr --with-python --with-geopm \
  && make -j \
  && make install

COPY requirements.txt /app/
RUN pip3 install -r /app/requirements.txt

COPY nrm-k3s.py /app/
ENV PYTHONPATH=/usr/lib/python3.10/site-packages:$PYTHONPATH
ENTRYPOINT ["nrmd"]