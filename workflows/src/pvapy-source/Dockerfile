####################################################################################################
# builder: install needed dependencies
####################################################################################################

FROM python:3.10-slim-bullseye AS builder

ENV PYTHONFAULTHANDLER=1 \
  PYTHONUNBUFFERED=1 \
  PYTHONHASHSEED=random \
  PIP_NO_CACHE_DIR=on \
  PIP_DISABLE_PIP_VERSION_CHECK=on \
  PIP_DEFAULT_TIMEOUT=100 \
  PYSETUP_PATH="/opt/pysetup"

RUN apt-get update \
    && apt-get install --no-install-recommends -y \
        curl \
        wget \
        # deps for building python deps
        build-essential \
    && apt-get install -y git \
    && apt-get clean && rm -rf /var/lib/apt/lists/* \
    \
    # install dumb-init
    && wget -O /dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.5/dumb-init_1.2.5_x86_64 \
    && chmod +x /dumb-init \
    && curl -sSL https://install.python-poetry.org | python3 -

####################################################################################################
# udf: used for running the udf vertices
####################################################################################################
FROM builder AS udf

ENV APP_PATH="/opt/app"

COPY requirements.txt $APP_PATH/requirements.txt
RUN pip3 install -U pip \
  && pip3 install -r $APP_PATH/requirements.txt

WORKDIR $APP_PATH
COPY ./ ./
 
RUN chmod +x $APP_PATH/entry.sh

ENTRYPOINT ["/dumb-init", "--"]
CMD ["sh", "-c", "$APP_PATH/entry.sh"]

EXPOSE 5000