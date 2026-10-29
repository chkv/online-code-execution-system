FROM ubuntu:24.04

RUN apt-get update && \
    apt-get install -y g++ && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd -g 1001 sandbox && \
    useradd -u 1001 -g 1001 -m -s /bin/bash sandbox

WORKDIR /workspace
RUN chmod 777 /workspace

USER sandbox

CMD ["g++", "--version"]
