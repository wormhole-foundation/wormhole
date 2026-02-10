FROM debian:bookworm-slim

# Install system dependencies
RUN apt-get update -qq && \
    apt-get install -y -qq curl jq ca-certificates git bsdmainutils && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install kubectl
RUN curl -LO "https://dl.k8s.io/release/v1.28.0/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/

# Install foundry
RUN curl -L https://foundry.paradigm.xyz | bash && \
    /root/.foundry/bin/foundryup

# Add foundry to PATH
ENV PATH="/root/.foundry/bin:$PATH"

# Create a simple health check
HealthCheck CMD echo "setup-ready"

# Default command
CMD ["/bin/bash"]