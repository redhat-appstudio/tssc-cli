FROM registry.access.redhat.com/ubi9/go-toolset:9.7-1763038106

ENV SHELLCHECK_VERSION=0.7.1
ENV YQ_VERSION=v4.25.2
ENV TEKTONCD_CLI_VERSION="v0.37.0"
ENV OC_VERSION=4.14.8

USER root

# hadolint ignore=DL3041
RUN dnf install -y -q --allowerasing --nobest \
    brotli \
    brotli-devel \
    ca-certificates \
    cmake \
    cpp \
    diffutils \
    gcc \
    gcc-c++ \
    jq \
    make \
    nodejs-devel \
    nodejs-libs \
    nodejs-packaging \
    openssl \
    openssl-devel \
    python3 \
    python3-pip \
    zlib \
    zlib-devel && \
    pip3 install --upgrade pip && \
    pip3 install --upgrade setuptools && \
    pip3 install yq yamllint && \
    dnf update -y && dnf clean all && \
    npm install -g yarn@1.22 npm@9 && \
    echo -n "node version: "; node -v; \
    echo -n "npm  version: "; npm -v; \
    echo -n "yarn version: "; yarn -v && \
    curl -fsSL --proto "=https" "https://github.com/koalaman/shellcheck/releases/download/v${SHELLCHECK_VERSION}/shellcheck-v${SHELLCHECK_VERSION}.linux.x86_64.tar.xz" | tar -xJvf - "shellcheck-v${SHELLCHECK_VERSION}/shellcheck" && \
    mv "shellcheck-v${SHELLCHECK_VERSION}/shellcheck" /usr/local/bin/shellcheck && \
    chmod 755 /usr/local/bin/shellcheck && \
    rm -rf "shellcheck-v${SHELLCHECK_VERSION}" && \
    curl -fsSL --proto "=https" -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 && \
    chmod 700 get_helm.sh && ./get_helm.sh && rm -rf get_helm.sh && \
    curl -fsSL --proto "=https" -o /usr/local/bin/yq "https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64" && chmod +x /usr/local/bin/yq && \
    curl -fsSL --proto "=https" -o tkn.tgz "https://github.com/tektoncd/cli/releases/download/${TEKTONCD_CLI_VERSION}/tkn_${TEKTONCD_CLI_VERSION:1}_Linux_x86_64.tar.gz" && tar -C "/usr/local/bin" --no-same-owner -xzf "tkn.tgz" tkn && rm -rf "tkn.tgz" && \
    curl -fsSL --proto "=https" -o argocd-linux-amd64 https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64 &&\
    install -m 555 argocd-linux-amd64 /usr/local/bin/argocd && \
    rm argocd-linux-amd64 && \
    curl -fsSL --proto "=https" -o /usr/local/bin/kubectl "https://dl.k8s.io/release/$(curl -sL --proto "=https" https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x /usr/local/bin/kubectl && \
    curl -fsSL --proto "=https" -o /usr/local/bin/cosign https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64 && \
    chmod +x /usr/local/bin/cosign && \
    curl -fsSL --proto "=https" https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin && \
    curl -fsSL --proto "=https" "https://mirror.openshift.com/pub/openshift-v4/clients/ocp/${OC_VERSION}/openshift-client-linux.tar.gz" -o /tmp/openshift-client-linux.tar.gz && \
    tar --no-same-owner -xzf /tmp/openshift-client-linux.tar.gz && \
    mv oc kubectl /usr/local/bin && \
    oc version --client && \
    kubectl version --client
