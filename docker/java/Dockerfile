FROM eclipse-temurin:21-jdk-alpine

# Create non-root user
RUN adduser -D -u 1000 sandbox

WORKDIR /workspace
RUN chmod 777 /workspace

USER 1000

CMD ["java", "--version"]
