FROM golang:latest

WORKDIR /usr/src/app

ENV SAMTOOLS_VERSION 1.14
ENV BCFTOOLS_VERSION 1.14

RUN apt-get update \
    && apt-get install --yes build-essential

RUN apt-get --yes install autoconf automake make gcc perl curl zlib1g-dev libbz2-dev liblzma-dev libcurl4-gnutls-dev libssl-dev libncurses5-dev

# Install samtools dependency
RUN cd /tmp \
    && wget https://github.com/samtools/samtools/releases/download/${SAMTOOLS_VERSION}/samtools-${SAMTOOLS_VERSION}.tar.bz2 \
    && tar xvjf samtools-${SAMTOOLS_VERSION}.tar.bz2 \
    && cd samtools-${SAMTOOLS_VERSION} \
    && ./configure --prefix=/usr/local \
    && make \
    && make install \
    && cd / && rm -rf /tmp/samtools-${SAMTOOLS_VERSION}

# Install bcftools dependency
RUN cd /tmp \
    && wget https://github.com/samtools/bcftools/releases/download/${BCFTOOLS_VERSION}/bcftools-${BCFTOOLS_VERSION}.tar.bz2 \
    && tar xvjf bcftools-${BCFTOOLS_VERSION}.tar.bz2 \
    && cd bcftools-${BCFTOOLS_VERSION} \
    && ./configure --prefix=/usr/local \
    && make \
    && make install \
    && cd / && rm -rf /tmp/bcftools-${BCFTOOLS_VERSION}

COPY go.mod go.sum ./
COPY cmd cmd
COPY internal internal
COPY data/config/umccr-s3.config.json ./
RUN mkdir temp
RUN go mod download

# Install htsget-refserver-utils dependency
RUN go mod download github.com/ga4gh/htsget-refserver-utils@v1.0.0 \
    && cd /go/pkg/mod/github.com/ga4gh/htsget-refserver-utils@v1.0.0 \
    && go install

ENV PATH="/usr/local:${PATH}"

RUN go build -o ./htsget-refserver ./cmd
EXPOSE 3000

CMD ["./htsget-refserver", "-config", "/usr/src/app/umccr-s3.config.json"]
