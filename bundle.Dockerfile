FROM scratch

# Core bundle labels.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=falcon-operator-rhmp
LABEL operators.operatorframework.io.bundle.channels.v1=alpha,certified-0.9
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-v1.30.0
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v4-alpha

# Labels for testing.
LABEL operators.operatorframework.io.test.mediatype.v1=scorecard+v1
LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/

# This block are standard Red Hat container labels
LABEL name="crowdstrike/falcon-operator-bundle" \
      License="ASL 2.0" \
      io.k8s.display-name="CrowdStrike Falcon Operator bundle" \
      io.k8s.description="CrowdStrike Falcon Operator's OLM bundle image" \
      summary="CrowdStrike Falcon Operator's OLM bundle image" \
      maintainer="CrowdStrike <integrations@crowdstrike.com>"

LABEL com.redhat.openshift.versions="v4.10-v4.15"
LABEL com.redhat.delivery.operator.bundle=true

# Copy files to locations specified by labels.
COPY bundle/manifests /manifests/
COPY bundle/metadata /metadata/
COPY bundle/tests/scorecard /tests/scorecard/
