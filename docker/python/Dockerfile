FROM python:3.12-slim

# Create non-root user for security
RUN groupadd -g 1000 sandbox && \
    useradd -u 1000 -g 1000 -m -s /bin/bash sandbox

WORKDIR /workspace
RUN chmod 777 /workspace

# Switch to non-root user
USER sandbox

CMD ["python3", "--version"]
