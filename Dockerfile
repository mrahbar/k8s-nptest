FROM debian:jessie
MAINTAINER Mahmoud Azad <mrahbar.azad@google.com>

# install binary and remove cache
RUN apt-get update \
    && apt-get install -y iperf3 curl wget net-tools gcc make \
    && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /tmp

# Download and build netperf from sources
RUN curl -LO https://github.com/HewlettPackard/netperf/archive/netperf-2.7.0.tar.gz && tar -xzf netperf-2.7.0.tar.gz
RUN cd netperf-netperf-2.7.0 && ./configure --prefix=/usr/local --bindir /usr/local/bin && make && make install

COPY nptests /usr/bin/
RUN chmod +x /usr/bin/nptests
ENTRYPOINT ["/usr/bin/nptests"]
