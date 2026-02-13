#
# Build
#

FROM registry.redhat.io/openshift4/ose-tools-rhel9@sha256:c1baccf320b0acaed693a07fd8df76758db0a38767ace30ccc79aed9ba8c4987 AS ose-tools
FROM registry.access.redhat.com/ubi10/go-toolset:1.25.3-1763633883 AS builder

ARG COMMIT_ID
ARG VERSION_ID

USER root
WORKDIR /workdir/tssc

COPY installer/ ./installer/

COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
COPY scripts/ ./scripts/
COPY test/ ./test/
COPY image/ ./image/
COPY vendor/ ./vendor/

COPY go.mod go.sum Makefile .goreleaser.yaml ./

RUN make test
RUN make GOFLAGS='-buildvcs=false' COMMIT_ID=${COMMIT_ID} VERSION=${VERSION_ID}

#
# Run
#

FROM registry.access.redhat.com/ubi10:10.1-1763341459

LABEL \
  name="tssc" \
  com.redhat.component="tssc" \
  description="Red Hat Trusted Software Supply Chain allows organizations to curate their own trusted, \
    repeatable pipelines that stay compliant with industry requirements. Built on proven, trusted open \
    source technologies, Red Hat Trusted Software Supply Chain is a set of solutions to protect users, \
    customers, and partners from risks and vulnerabilities in their software factory." \
  io.k8s.description="Red Hat Trusted Software Supply Chain allows organizations to curate their own trusted, \
    repeatable pipelines that stay compliant with industry requirements. Built on proven, trusted open \
    source technologies, Red Hat Trusted Software Supply Chain is a set of solutions to protect users, \
    customers, and partners from risks and vulnerabilities in their software factory." \
  summary="Provides the tssc binary." \
  io.k8s.display-name="Red Hat Trusted Software Supply Chain CLI" \
  io.openshift.tags="tssc tas tpa rhdh ec tap openshift"

# Banner
RUN echo 'cat << "EOF"' >> /etc/profile && \
    echo '╔═══════════════════════════════════════════════════════╗' >> /etc/profile && \
    echo '║   Welcome to the Trusted Software Factory Installer   ║' >> /etc/profile && \
    echo '╚═══════════════════════════════════════════════════════╝' >> /etc/profile && \
    echo ' ' >> /etc/profile && \
    echo 'To deploy the Trusted Software Factory:' >> /etc/profile && \
    echo '  - Login to the cluster' >> /etc/profile && \
    echo '  - Create the TSF config on the cluster' >> /etc/profile && \
    echo '  - Create the integrations' >> /etc/profile && \
    echo '  - Deploy TSF' >> /etc/profile && \
    echo ' ' >> /etc/profile && \
    echo 'For more information, please visit https://github.com/redhat-appstudio/tssc-cli/blob/tsf/docs/trusted-software-factory.md' >> /etc/profile && \
    echo ' ' >> /etc/profile && \
    echo 'EOF' >> /etc/profile

WORKDIR /licenses

COPY LICENSE.txt .

WORKDIR /tssc

COPY --from=ose-tools /usr/bin/jq /usr/bin/kubectl /usr/bin/oc /usr/bin/vi /usr/bin/
# jq libraries
COPY --from=ose-tools /usr/lib64/libjq.so.1 /usr/lib64/libonig.so.5 /usr/lib64/
# vi libraries
COPY --from=ose-tools /usr/libexec/vi /usr/libexec/

COPY --from=builder /workdir/tssc/installer/charts ./charts
COPY --from=builder /workdir/tssc/installer/config.yaml ./
COPY --from=builder /workdir/tssc/bin/tssc /usr/local/bin/tssc
COPY --from=builder /workdir/tssc/scripts/ ./scripts/

RUN groupadd --gid 9999 -r tssc && \
    useradd -r -d /tssc -g tssc -s /sbin/nologin --uid 9999 tssc && \
    chown -R tssc:tssc .

USER tssc

RUN echo "# jq" && jq --version && \
    echo "# kubectl" && kubectl version --client && \
    echo "# oc" && oc version

ENV KUBECONFIG=/tssc/.kube/config

ENTRYPOINT ["tssc"]
