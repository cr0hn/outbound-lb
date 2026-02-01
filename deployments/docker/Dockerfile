# Runtime image - binary is pre-compiled by GoReleaser
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy pre-compiled binary from GoReleaser
COPY outbound-lb /app/outbound-lb

# Set ownership
RUN chown -R appuser:appuser /app

USER appuser

# Expose ports
EXPOSE 3128 9090

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9090/health || exit 1

ENTRYPOINT ["/app/outbound-lb"]
