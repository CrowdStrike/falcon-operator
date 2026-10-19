"""
Tests for migrate-to-clusterguard.py

Each test covers a distinct field mapping or behavioral concern so that a
regression in any single mapping produces a focused, easy-to-diagnose failure.
"""

import sys
import os
import pytest

from ruamel.yaml import YAML
from ruamel.yaml.comments import CommentedMap

sys.path.insert(0, os.path.dirname(__file__))
from migrate_to_clusterguard import (
    migrate, migrate_node_sensor, migrate_admission, migrate_image_analyzer,
    load_files, assemble, apply_overrides, main,
)


# ── helpers ───────────────────────────────────────────────────────────────────

yaml = YAML()


def load(text: str) -> CommentedMap:
    return yaml.load(text)


def make_deployment(spec_yaml: str = "") -> CommentedMap:
    base = f"""\
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconDeployment
metadata:
  name: falcon
spec:
{spec_yaml}
"""
    return load(base)


def run(spec_yaml: str = "") -> tuple[CommentedMap, list]:
    doc = make_deployment(spec_yaml)
    warnings = []
    migrate(doc, warnings)
    return doc["spec"], warnings


# ── deploy flags ──────────────────────────────────────────────────────────────

def test_sets_deploy_cluster_guard_true():
    spec, _ = run()
    assert spec["deployClusterGuard"] is True


def test_sets_deploy_node_sensor_false():
    spec, _ = run("  deployNodeSensor: true")
    assert spec["deployNodeSensor"] is False


def test_node_sensor_enabled_false_when_deploy_node_sensor_false():
    spec, _ = run("  deployNodeSensor: false")
    assert spec["falconClusterGuard"]["nodeSensor"]["enabled"] is False


def test_node_sensor_enabled_false_when_no_node_sensor_in_input():
    result = assemble(make_deployment(""), {}, "falcon", [])
    assert result["spec"]["falconClusterGuard"]["nodeSensor"]["enabled"] is False


def test_node_sensor_enabled_not_set_when_deploy_node_sensor_true():
    spec, _ = run("""\
  deployNodeSensor: true
  falconNodeSensor:
    installNamespace: falcon-system
""")
    assert spec["falconClusterGuard"].get("nodeSensor", {}).get("enabled") is not False


def test_sets_deploy_admission_false():
    spec, _ = run("  deployAdmissionController: true")
    assert spec["deployAdmissionController"] is False


def test_wrong_kind_skipped():
    doc = load("kind: FalconNodeSensor\nmetadata:\n  name: x")
    warnings = []
    migrate(doc, warnings)
    assert any("skipping" in w for w in warnings)
    assert "deployClusterGuard" not in doc


# ── node sensor: top-level promotion ─────────────────────────────────────────

def test_node_sensor_install_namespace_promoted():
    spec, _ = run("""\
  falconNodeSensor:
    installNamespace: falcon-system
""")
    assert spec["falconClusterGuard"]["installNamespace"] == "falcon-system"


def test_node_sensor_falcon_api_promoted():
    spec, _ = run("""\
  falconNodeSensor:
    falcon_api:
      client_id: abc
      client_secret: xyz
""")
    assert spec["falconClusterGuard"]["falcon_api"]["client_id"] == "abc"


def test_node_sensor_falcon_promoted():
    spec, _ = run("""\
  falconNodeSensor:
    falcon:
      cid: ABCD-1
""")
    assert spec["falconClusterGuard"]["falcon"]["cid"] == "ABCD-1"


def test_node_sensor_falcon_secret_promoted():
    spec, _ = run("""\
  falconNodeSensor:
    falconSecret:
      enabled: true
""")
    assert spec["falconClusterGuard"]["falconSecret"]["enabled"] is True


# ── node sensor: node.* fields ────────────────────────────────────────────────

def test_node_image_promoted_to_fcg():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      image: my-registry/falcon:latest
""")
    assert spec["falconClusterGuard"]["image"] == "my-registry/falcon:latest"


def test_node_image_pull_policy_promoted_to_fcg():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      imagePullPolicy: Always
""")
    assert spec["falconClusterGuard"]["imagePullPolicy"] == "Always"


def test_node_image_pull_secrets_promoted_to_fcg():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      imagePullSecrets:
        - name: my-secret
""")
    assert spec["falconClusterGuard"]["imagePullSecrets"][0]["name"] == "my-secret"


def test_node_tolerations_in_node_sensor():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      tolerations:
        - key: node-role.kubernetes.io/master
          operator: Exists
          effect: NoSchedule
""")
    tols = spec["falconClusterGuard"]["nodeSensor"]["tolerations"]
    assert tols[0]["key"] == "node-role.kubernetes.io/master"


def test_node_affinity_in_node_sensor():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms: []
""")
    assert "nodeAffinity" in spec["falconClusterGuard"]["nodeSensor"]


def test_node_update_strategy_in_node_sensor():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      updateStrategy:
        rollingUpdate:
          maxUnavailable: 1
""")
    assert "updateStrategy" in spec["falconClusterGuard"]["nodeSensor"]


def test_node_termination_grace_period_in_node_sensor():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      terminationGracePeriod: 90
""")
    assert spec["falconClusterGuard"]["nodeSensor"]["terminationGracePeriod"] == 90


def test_node_service_account_in_node_sensor():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      serviceAccount:
        annotations:
          eks.amazonaws.com/role-arn: arn:aws:iam::123:role/role
""")
    assert "serviceAccount" in spec["falconClusterGuard"]["nodeSensor"]


def test_node_disable_cleanup_in_node_sensor():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      disableCleanup: true
""")
    assert spec["falconClusterGuard"]["nodeSensor"]["disableCleanup"] is True


def test_node_resources_in_node_sensor():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      resources:
        limits:
          cpu: 750m
""")
    assert spec["falconClusterGuard"]["nodeSensor"]["resources"]["limits"]["cpu"] == "750m"


def test_node_backend_in_node_sensor():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      backend: bpf
""")
    assert spec["falconClusterGuard"]["nodeSensor"]["backend"] == "bpf"


def test_node_gke_in_node_sensor():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      gke:
        autopilot: true
""")
    assert "gke" in spec["falconClusterGuard"]["nodeSensor"]


def test_node_priority_class_in_node_sensor():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      priorityClass:
        enabled: true
        name: high-priority
""")
    assert "priorityClass" in spec["falconClusterGuard"]["nodeSensor"]


def test_node_advanced_in_node_sensor():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      advanced:
        foo: bar
""")
    assert "advanced" in spec["falconClusterGuard"]["nodeSensor"]


def test_node_cluster_name_in_node_sensor():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      clusterName: my-cluster
""")
    assert spec["falconClusterGuard"]["nodeSensor"]["clusterName"] == "my-cluster"


def test_node_internal_skipped_with_warning():
    _, warnings = run("""\
  falconNodeSensor:
    internal:
      foo: bar
""")
    assert any("internal" in w for w in warnings)


# ── admission: top-level promotion ───────────────────────────────────────────

def test_admission_install_namespace_promoted():
    spec, _ = run("""\
  falconAdmission:
    installNamespace: falcon-kac
""")
    assert spec["falconClusterGuard"]["installNamespace"] == "falcon-kac"


def test_admission_falcon_api_promoted():
    spec, _ = run("""\
  falconAdmission:
    falcon_api:
      client_id: def
""")
    assert spec["falconClusterGuard"]["falcon_api"]["client_id"] == "def"


def test_admission_image_promoted():
    spec, _ = run("""\
  falconAdmission:
    image: my-registry/kac:latest
""")
    assert spec["falconClusterGuard"]["image"] == "my-registry/kac:latest"


def test_admission_registry_crowdstrike_stays_crowdstrike():
    spec, warnings = run("""\
  falconAdmission:
    registry:
      type: crowdstrike
""")
    assert spec["falconClusterGuard"]["registry"]["type"] == "crowdstrike"
    assert not any("mapped to" in w for w in warnings)


def test_admission_registry_acr_mapped_to_private():
    spec, warnings = run("""\
  falconAdmission:
    registry:
      type: acr
      acr_name: my-acr
""")
    assert spec["falconClusterGuard"]["registry"]["type"] == "private"
    assert any("acr" in w and "mapped to 'private'" in w for w in warnings)
    assert any("acr_name" in w for w in warnings)


def test_admission_registry_ecr_mapped_to_private():
    spec, warnings = run("""\
  falconAdmission:
    registry:
      type: ecr
""")
    assert spec["falconClusterGuard"]["registry"]["type"] == "private"
    assert any("ecr" in w and "mapped to 'private'" in w for w in warnings)


def test_admission_registry_gcr_mapped_to_private():
    spec, warnings = run("""\
  falconAdmission:
    registry:
      type: gcr
""")
    assert spec["falconClusterGuard"]["registry"]["type"] == "private"
    assert any("gcr" in w and "mapped to 'private'" in w for w in warnings)


def test_admission_registry_openshift_mapped_to_private():
    spec, warnings = run("""\
  falconAdmission:
    registry:
      type: openshift
""")
    assert spec["falconClusterGuard"]["registry"]["type"] == "private"
    assert any("openshift" in w and "mapped to 'private'" in w for w in warnings)


def test_admission_registry_tls_preserved():
    spec, _ = run("""\
  falconAdmission:
    registry:
      type: ecr
      tls:
        insecure_skip_verify: true
""")
    assert spec["falconClusterGuard"]["registry"]["tls"]["insecure_skip_verify"] is True


def test_admission_registry_acr_name_skipped_with_warning():
    _, warnings = run("""\
  falconAdmission:
    registry:
      type: acr
      acr_name: my-acr
""")
    assert any("acr_name" in w and "no equivalent" in w for w in warnings)


def test_admission_registry_not_overwritten_by_admission():
    spec, warnings = run("""\
  falconNodeSensor:
    installNamespace: falcon-system
  falconAdmission:
    registry:
      type: ecr
""")
    # node_sensor runs first but doesn't set registry, so admission should set it
    assert spec["falconClusterGuard"]["registry"]["type"] == "private"

    # But if FCG registry already exists, it should warn
    spec2, warnings2 = run("""\
  falconAdmission:
    registry:
      type: ecr
""")
    # Manually pre-set registry on fcg to test the skip path
    from migrate_to_clusterguard import migrate, _translate_registry
    doc = make_deployment("""\
  falconAdmission:
    registry:
      type: ecr
""")
    doc["spec"].setdefault("falconClusterGuard", CommentedMap())["registry"] = CommentedMap([("type", "crowdstrike")])
    w = []
    migrate(doc, w)
    assert doc["spec"]["falconClusterGuard"]["registry"]["type"] == "crowdstrike"
    assert any("already set" in x for x in w)


def test_admission_resourcequota_skipped_with_warning():
    _, warnings = run("""\
  falconAdmission:
    resourcequota:
      pods: "2"
""")
    assert any("resourcequota" in w for w in warnings)


def test_admission_cluster_name_in_admission_config():
    spec, _ = run("""\
  falconAdmission:
    clusterName: my-cluster
""")
    assert spec["falconClusterGuard"]["controller"]["clusterName"] == "my-cluster"


# ── admission: admissionConfig.* fields that promote to FCG top-level ────────

def test_admission_config_imagepullpolicy_promoted_to_fcg():
    """admissionConfig.imagePullPolicy does not exist on FalconClusterGuardAdmissionSpec;
    it must go to falconClusterGuard.imagePullPolicy."""
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      imagePullPolicy: Always
""")
    assert spec["falconClusterGuard"]["imagePullPolicy"] == "Always"
    assert "imagePullPolicy" not in spec["falconClusterGuard"].get("controller", {})


def test_admission_config_imagepullsecrets_promoted_to_fcg():
    """admissionConfig.imagePullSecrets does not exist on FalconClusterGuardAdmissionSpec;
    it must go to falconClusterGuard.imagePullSecrets."""
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      imagePullSecrets:
        - name: kac-secret
""")
    assert spec["falconClusterGuard"]["imagePullSecrets"][0]["name"] == "kac-secret"
    assert "imagePullSecrets" not in spec["falconClusterGuard"].get("controller", {})


def test_admission_config_imagepullpolicy_does_not_overwrite_node_sensor():
    """Node sensor sets imagePullPolicy first; admissionConfig must not overwrite it."""
    spec, _ = run("""\
  falconNodeSensor:
    node:
      imagePullPolicy: IfNotPresent
  falconAdmission:
    admissionConfig:
      imagePullPolicy: Always
""")
    assert spec["falconClusterGuard"]["imagePullPolicy"] == "IfNotPresent"


# ── admission: admissionConfig.* fields that stay in admissionConfig ─────────

def test_admission_config_service_account():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      serviceAccount:
        annotations:
          eks.amazonaws.com/role-arn: arn:aws:iam::123:role/role
""")
    assert "serviceAccount" in spec["falconClusterGuard"]["controller"]


def test_admission_config_service_port():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      servicePort: 8443
""")
    assert spec["falconClusterGuard"]["controller"]["servicePort"] == 8443


def test_admission_config_container_port():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      containerPort: 4443
""")
    assert spec["falconClusterGuard"]["controller"]["containerPort"] == 4443


def test_admission_config_tls():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      tls:
        validity: 365
""")
    assert "tls" in spec["falconClusterGuard"]["controller"]


def test_admission_config_failure_policy():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      failurePolicy: Ignore
""")
    assert spec["falconClusterGuard"]["controller"]["failurePolicy"] == "Ignore"


def test_admission_config_disabled_namespaces():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      disabledNamespaces:
        namespaces:
          - kube-system
""")
    ns = spec["falconClusterGuard"]["controller"]["disabledNamespaces"]["namespaces"]
    assert "kube-system" in ns


def test_admission_config_watcher_enabled():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      watcherEnabled: true
""")
    assert spec["falconClusterGuard"]["controller"]["watcherEnabled"] is True


def test_admission_config_snapshots_enabled():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      snapshotsEnabled: false
""")
    assert spec["falconClusterGuard"]["controller"]["snapshotsEnabled"] is False


def test_admission_config_snapshots_interval():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      snapshotsInterval: 12h
""")
    assert spec["falconClusterGuard"]["controller"]["snapshotsInterval"] == "12h"


def test_admission_config_admission_control_enabled():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      admissionControlEnabled: false
""")
    assert spec["falconClusterGuard"]["controller"]["admissionControlEnabled"] is False


def test_admission_config_configmap_watcher_enabled():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      configMapWatcherEnabled: false
""")
    assert spec["falconClusterGuard"]["controller"]["configMapWatcherEnabled"] is False


def test_admission_config_falcon_image_analyzer_namespace():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      falconImageAnalyzerNamespace: falcon-iar
""")
    assert spec["falconClusterGuard"]["controller"]["falconImageAnalyzerNamespace"] == "falcon-iar"


def test_admission_config_replicas():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      replicas: 3
""")
    assert spec["falconClusterGuard"]["controller"]["replicas"] == 3


def test_admission_config_node_affinity():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms: []
""")
    assert "nodeAffinity" in spec["falconClusterGuard"]["controller"]


def test_admission_config_tolerations():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      tolerations:
        - key: node-role.kubernetes.io/master
          operator: Exists
          effect: NoSchedule
""")
    tols = spec["falconClusterGuard"]["controller"]["tolerations"]
    assert tols[0]["key"] == "node-role.kubernetes.io/master"


def test_admission_config_update_strategy():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      updateStrategy:
        rollingUpdate:
          maxUnavailable: 1
""")
    assert "updateStrategy" in spec["falconClusterGuard"]["controller"]


def test_admission_config_resources():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      resources:
        limits:
          cpu: 500m
""")
    assert spec["falconClusterGuard"]["controller"]["resources"]["limits"]["cpu"] == "500m"


def test_admission_config_resources_client():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      resourcesClient:
        limits:
          memory: 256Mi
""")
    assert spec["falconClusterGuard"]["controller"]["resourcesClient"]["limits"]["memory"] == "256Mi"


def test_admission_config_resources_client_no_webhook():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      resourcesClientNoWebhook:
        limits:
          cpu: 250m
""")
    assert spec["falconClusterGuard"]["controller"]["resourcesClientNoWebhook"]["limits"]["cpu"] == "250m"


def test_admission_config_resources_watcher():
    spec, _ = run("""\
  falconAdmission:
    admissionConfig:
      resourcesWatcher:
        requests:
          memory: 128Mi
""")
    assert spec["falconClusterGuard"]["controller"]["resourcesWatcher"]["requests"]["memory"] == "128Mi"


# ── conflict resolution ───────────────────────────────────────────────────────

def test_node_sensor_wins_install_namespace_conflict():
    """Node sensor sets installNamespace first; admission must not overwrite it."""
    spec, _ = run("""\
  falconNodeSensor:
    installNamespace: falcon-system
  falconAdmission:
    installNamespace: falcon-kac
""")
    assert spec["falconClusterGuard"]["installNamespace"] == "falcon-system"


def test_node_sensor_wins_falcon_api_conflict():
    spec, _ = run("""\
  falconNodeSensor:
    falcon_api:
      client_id: from-node
  falconAdmission:
    falcon_api:
      client_id: from-admission
""")
    assert spec["falconClusterGuard"]["falcon_api"]["client_id"] == "from-node"


def test_node_sensor_wins_image_conflict():
    spec, _ = run("""\
  falconNodeSensor:
    node:
      image: node-image:1.0
  falconAdmission:
    image: admission-image:1.0
""")
    assert spec["falconClusterGuard"]["image"] == "node-image:1.0"


# ── deep copy (no anchors) ────────────────────────────────────────────────────

def test_no_yaml_anchors_in_output():
    doc = make_deployment("""\
  falconNodeSensor:
    installNamespace: falcon-system
    falcon_api:
      client_id: abc
  falconAdmission:
    admissionConfig:
      failurePolicy: Ignore
""")
    warnings = []
    migrate(doc, warnings)

    import io
    buf = io.StringIO()
    yaml.dump(doc, buf)
    output = buf.getvalue()

    assert "&id" not in output
    assert "*id" not in output


# ── load_files ────────────────────────────────────────────────────────────────

def test_load_files_single_node_sensor(tmp_path):
    p = tmp_path / "node.yaml"
    _yaml = YAML()
    _yaml.preserve_quotes = True
    cr = CommentedMap()
    cr["apiVersion"] = "falcon.crowdstrike.com/v1alpha1"
    cr["kind"] = "FalconNodeSensor"
    cr["metadata"] = CommentedMap([("name", "test")])
    cr["spec"] = CommentedMap([("installNamespace", "falcon-system")])
    with open(p, "w") as f:
        _yaml.dump(cr, f)

    fd_doc, crs = load_files([str(p)], [])
    assert fd_doc is None
    assert "FalconNodeSensor" in crs
    assert crs["FalconNodeSensor"]["installNamespace"] == "falcon-system"


def test_load_files_single_falcon_deployment(tmp_path):
    p = tmp_path / "fd.yaml"
    _yaml = YAML()
    _yaml.preserve_quotes = True
    fd = CommentedMap()
    fd["apiVersion"] = "falcon.crowdstrike.com/v1alpha1"
    fd["kind"] = "FalconDeployment"
    fd["metadata"] = CommentedMap([("name", "falcon")])
    fd["spec"] = CommentedMap([("deployNodeSensor", True)])
    with open(p, "w") as f:
        _yaml.dump(fd, f)

    fd_doc, crs = load_files([str(p)], [])
    assert fd_doc is not None
    assert fd_doc["kind"] == "FalconDeployment"
    assert crs == {}


def test_load_files_fd_plus_node_sensor(tmp_path):
    _yaml = YAML()
    _yaml.preserve_quotes = True

    fd_path = tmp_path / "fd.yaml"
    fd = CommentedMap()
    fd["apiVersion"] = "falcon.crowdstrike.com/v1alpha1"
    fd["kind"] = "FalconDeployment"
    fd["metadata"] = CommentedMap([("name", "falcon")])
    fd["spec"] = CommentedMap()
    with open(fd_path, "w") as f:
        _yaml.dump(fd, f)

    ns_path = tmp_path / "node.yaml"
    ns = CommentedMap()
    ns["apiVersion"] = "falcon.crowdstrike.com/v1alpha1"
    ns["kind"] = "FalconNodeSensor"
    ns["metadata"] = CommentedMap([("name", "test")])
    ns["spec"] = CommentedMap([("installNamespace", "falcon-system")])
    with open(ns_path, "w") as f:
        _yaml.dump(ns, f)

    fd_doc, crs = load_files([str(fd_path), str(ns_path)], [])
    assert fd_doc is not None
    assert "FalconNodeSensor" in crs


def test_load_files_duplicate_fd_exits(tmp_path):
    _yaml = YAML()
    _yaml.preserve_quotes = True
    for name in ("a.yaml", "b.yaml"):
        p = tmp_path / name
        fd = CommentedMap()
        fd["apiVersion"] = "falcon.crowdstrike.com/v1alpha1"
        fd["kind"] = "FalconDeployment"
        fd["metadata"] = CommentedMap([("name", "falcon")])
        fd["spec"] = CommentedMap()
        with open(p, "w") as f:
            _yaml.dump(fd, f)

    with pytest.raises(SystemExit):
        load_files([str(tmp_path / "a.yaml"), str(tmp_path / "b.yaml")], [])


def test_load_files_duplicate_crd_exits(tmp_path):
    _yaml = YAML()
    _yaml.preserve_quotes = True
    for name in ("a.yaml", "b.yaml"):
        p = tmp_path / name
        cr = CommentedMap()
        cr["apiVersion"] = "falcon.crowdstrike.com/v1alpha1"
        cr["kind"] = "FalconNodeSensor"
        cr["metadata"] = CommentedMap([("name", "test")])
        cr["spec"] = CommentedMap()
        with open(p, "w") as f:
            _yaml.dump(cr, f)

    with pytest.raises(SystemExit):
        load_files([str(tmp_path / "a.yaml"), str(tmp_path / "b.yaml")], [])


def test_load_files_unrecognized_kind_warns(tmp_path):
    p = tmp_path / "cm.yaml"
    p.write_text("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n")
    warnings = []
    with pytest.raises(SystemExit):
        load_files([str(p)], warnings)
    assert any("unrecognized" in w for w in warnings)


def test_load_files_file_not_found_exits():
    with pytest.raises(SystemExit):
        load_files(["/no/such/file.yaml"], [])


# ── assemble (precedence) ────────────────────────────────────────────────────

def _make_fd_doc(spec_yaml: str = "") -> CommentedMap:
    """Build a minimal FalconDeployment document for assemble() tests."""
    base = f"""\
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconDeployment
metadata:
  name: falcon
spec:
{spec_yaml}
"""
    return yaml.load(base)


def _make_cr_spec(spec_yaml: str) -> CommentedMap:
    """Parse a YAML snippet into a CommentedMap representing a CR spec."""
    return yaml.load(spec_yaml) or CommentedMap()


def test_assemble_fd_deploy_true_ignores_standalone_crd():
    fd_doc = _make_fd_doc("  deployNodeSensor: true\n  falconNodeSensor:\n    installNamespace: from-fd")
    crs = {"FalconNodeSensor": _make_cr_spec("installNamespace: from-standalone")}
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    assert result["spec"]["falconClusterGuard"]["installNamespace"] == "from-fd"
    assert any("ignored" in w.lower() and "FalconNodeSensor" in w for w in warnings)


def test_assemble_fd_deploy_false_accepts_standalone_crd():
    fd_doc = _make_fd_doc("  deployNodeSensor: false")
    crs = {"FalconNodeSensor": _make_cr_spec("installNamespace: from-standalone")}
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    assert result["spec"]["falconClusterGuard"]["installNamespace"] == "from-standalone"


def test_assemble_fd_deploy_absent_accepts_standalone_crd():
    fd_doc = _make_fd_doc("")
    crs = {"FalconNodeSensor": _make_cr_spec("installNamespace: from-standalone")}
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    assert result["spec"]["falconClusterGuard"]["installNamespace"] == "from-standalone"


def test_assemble_fd_shared_field_wins_over_crd():
    fd_doc = _make_fd_doc("  falcon_api:\n    client_id: from-fd")
    crs = {"FalconNodeSensor": _make_cr_spec("falcon_api:\n  client_id: from-standalone\ninstallNamespace: ns")}
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    assert result["spec"]["falcon_api"]["client_id"] == "from-fd"


def test_assemble_no_fd_creates_from_crds():
    crs = {"FalconNodeSensor": _make_cr_spec("installNamespace: falcon-system")}
    warnings = []
    result = assemble(None, crs, "my-falcon", warnings)
    assert result["kind"] == "FalconDeployment"
    assert result["metadata"]["name"] == "my-falcon"
    assert result["spec"]["falconClusterGuard"]["installNamespace"] == "falcon-system"


def test_assemble_fd_only_no_crds():
    fd_doc = _make_fd_doc("  deployNodeSensor: true\n  falconNodeSensor:\n    installNamespace: ns")
    warnings = []
    result = assemble(fd_doc, {}, "falcon", warnings)
    assert result["spec"]["falconClusterGuard"]["installNamespace"] == "ns"


def test_assemble_fd_admission_true_ignores_standalone():
    fd_doc = _make_fd_doc("  deployAdmissionController: true\n  falconAdmission:\n    installNamespace: from-fd")
    crs = {"FalconAdmission": _make_cr_spec("installNamespace: from-standalone")}
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    assert result["spec"]["falconClusterGuard"]["installNamespace"] == "from-fd"
    assert any("ignored" in w.lower() and "FalconAdmission" in w for w in warnings)


def test_assemble_fd_container_true_ignores_standalone():
    fd_doc = _make_fd_doc("  deployContainerSensor: true\n  falconContainerSensor:\n    injector:\n      listenPort: 4433")
    crs = {"FalconContainer": _make_cr_spec("injector:\n  listenPort: 9999")}
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    assert result["spec"]["falconContainerSensor"]["injector"]["listenPort"] == 4433
    assert any("ignored" in w.lower() and "FalconContainer" in w for w in warnings)


def test_assemble_absent_kinds_get_false_flags():
    crs = {"FalconNodeSensor": _make_cr_spec("installNamespace: ns")}
    warnings = []
    result = assemble(None, crs, "falcon", warnings)
    assert result["spec"]["deployContainerSensor"] is False
    assert result["spec"]["deployImageAnalyzer"] is False


# ── apply_overrides ──────────────────────────────────────────────────────────

def test_fcg_image_override_sets_image_and_private_registry():
    fd_doc = _make_fd_doc("  deployNodeSensor: false")
    crs = {"FalconNodeSensor": _make_cr_spec("installNamespace: ns")}
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    apply_overrides(result, fcg_image_override="my-registry/fcg:1.0", fcg_image_pull_policy=None)
    fcg = result["spec"]["falconClusterGuard"]
    assert fcg["image"] == "my-registry/fcg:1.0"
    assert fcg["registry"]["type"] == "private"


def test_fcg_image_override_replaces_migrated_image():
    fd_doc = _make_fd_doc("")
    crs = {"FalconNodeSensor": _make_cr_spec("node:\n  image: old-image:1.0\ninstallNamespace: ns")}
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    apply_overrides(result, fcg_image_override="new-image:2.0", fcg_image_pull_policy=None)
    assert result["spec"]["falconClusterGuard"]["image"] == "new-image:2.0"
    assert result["spec"]["falconClusterGuard"]["registry"]["type"] == "private"


def test_no_fcg_image_override_leaves_image_untouched():
    fd_doc = _make_fd_doc("")
    crs = {"FalconNodeSensor": _make_cr_spec("node:\n  image: original:1.0\ninstallNamespace: ns")}
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    apply_overrides(result, fcg_image_override=None, fcg_image_pull_policy=None)
    assert result["spec"]["falconClusterGuard"]["image"] == "original:1.0"


def test_fcg_image_pull_policy_override():
    fd_doc = _make_fd_doc("")
    crs = {"FalconNodeSensor": _make_cr_spec("installNamespace: ns")}
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    apply_overrides(result, fcg_image_override=None, fcg_image_pull_policy="Always")
    assert result["spec"]["falconClusterGuard"]["imagePullPolicy"] == "Always"


def test_fcg_image_pull_policy_replaces_migrated():
    fd_doc = _make_fd_doc("")
    crs = {"FalconNodeSensor": _make_cr_spec("node:\n  imagePullPolicy: IfNotPresent\ninstallNamespace: ns")}
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    apply_overrides(result, fcg_image_override=None, fcg_image_pull_policy="Always")
    assert result["spec"]["falconClusterGuard"]["imagePullPolicy"] == "Always"


def test_no_fcg_image_pull_policy_leaves_untouched():
    fd_doc = _make_fd_doc("")
    crs = {"FalconNodeSensor": _make_cr_spec("node:\n  imagePullPolicy: IfNotPresent\ninstallNamespace: ns")}
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    apply_overrides(result, fcg_image_override=None, fcg_image_pull_policy=None)
    assert result["spec"]["falconClusterGuard"]["imagePullPolicy"] == "IfNotPresent"


def test_both_overrides_applied_together():
    fd_doc = _make_fd_doc("")
    crs = {"FalconNodeSensor": _make_cr_spec("installNamespace: ns")}
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    apply_overrides(result, fcg_image_override="my-image:1.0", fcg_image_pull_policy="Never")
    fcg = result["spec"]["falconClusterGuard"]
    assert fcg["image"] == "my-image:1.0"
    assert fcg["registry"]["type"] == "private"
    assert fcg["imagePullPolicy"] == "Never"


def test_apply_overrides_admission_control_false():
    fd_doc = _make_fd_doc("")
    crs = {"FalconNodeSensor": _make_cr_spec("installNamespace: ns")}
    result = assemble(fd_doc, crs, "falcon", [])
    apply_overrides(result, None, None, admission_control=False)
    assert result["spec"]["falconClusterGuard"]["controller"]["admissionControlEnabled"] is False


def test_apply_overrides_admission_control_true():
    fd_doc = _make_fd_doc("")
    crs = {"FalconNodeSensor": _make_cr_spec("installNamespace: ns")}
    result = assemble(fd_doc, crs, "falcon", [])
    apply_overrides(result, None, None, admission_control=True)
    assert result["spec"]["falconClusterGuard"]["controller"]["admissionControlEnabled"] is True


def test_apply_overrides_admission_control_overrides_migrated():
    """CLI --admission-control wins over value already migrated from admissionConfig."""
    fd_doc = _make_fd_doc("")
    crs = {"FalconAdmission": _make_cr_spec("admissionConfig:\n  admissionControlEnabled: true")}
    result = assemble(fd_doc, crs, "falcon", [])
    apply_overrides(result, None, None, admission_control=False)
    assert result["spec"]["falconClusterGuard"]["controller"]["admissionControlEnabled"] is False


def test_apply_overrides_image_analyzer_true():
    fd_doc = _make_fd_doc("")
    crs = {"FalconNodeSensor": _make_cr_spec("installNamespace: ns")}
    result = assemble(fd_doc, crs, "falcon", [])
    apply_overrides(result, None, None, image_analyzer=True)
    assert result["spec"]["falconClusterGuard"]["imageAnalyzer"]["enabled"] is True


def test_apply_overrides_image_analyzer_false():
    fd_doc = _make_fd_doc("")
    crs = {"FalconImageAnalyzer": _make_cr_spec("installNamespace: ns")}
    result = assemble(fd_doc, crs, "falcon", [])
    apply_overrides(result, None, None, image_analyzer=False)
    assert result["spec"]["falconClusterGuard"]["imageAnalyzer"]["enabled"] is False


# ── image analyzer migration ─────────────────────────────────────────────────

def _run_with_iar(spec_yaml: str = "") -> tuple[CommentedMap, list]:
    """Helper: build a FD with falconImageAnalyzer and run migrate()."""
    doc = make_deployment(spec_yaml)
    warnings = []
    migrate(doc, warnings)
    return doc["spec"], warnings


def test_deploy_image_analyzer_set_false_after_migration():
    spec, _ = _run_with_iar("  deployImageAnalyzer: true\n  falconImageAnalyzer:\n    installNamespace: ns")
    assert spec["deployImageAnalyzer"] is False


def test_image_analyzer_enabled_true_when_deploy_flag_was_true():
    spec, _ = _run_with_iar("  deployImageAnalyzer: true\n  falconImageAnalyzer:\n    installNamespace: ns")
    assert spec["falconClusterGuard"]["imageAnalyzer"]["enabled"] is True


def test_image_analyzer_enabled_true_when_inferred_from_sub_spec():
    spec, _ = _run_with_iar("  falconImageAnalyzer:\n    installNamespace: ns")
    assert spec["falconClusterGuard"]["imageAnalyzer"]["enabled"] is True


def test_image_analyzer_enabled_not_set_when_deploy_flag_false():
    spec, _ = _run_with_iar("  deployImageAnalyzer: false")
    assert spec["falconClusterGuard"].get("imageAnalyzer", {}).get("enabled") is not True


def test_image_analyzer_sub_spec_removed_from_fd_after_migration():
    spec, _ = _run_with_iar("  falconImageAnalyzer:\n    installNamespace: ns")
    assert "falconImageAnalyzer" not in spec


# migrate_image_analyzer: top-level promotions

def test_image_analyzer_install_namespace_promoted():
    spec, _ = _run_with_iar("  deployImageAnalyzer: true\n  falconImageAnalyzer:\n    installNamespace: falcon-iar")
    assert spec["falconClusterGuard"]["installNamespace"] == "falcon-iar"


def test_image_analyzer_falcon_api_promoted():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    falcon_api:\n"
        "      client_id: iar-client\n"
    )
    assert spec["falconClusterGuard"]["falcon_api"]["client_id"] == "iar-client"


def test_image_analyzer_falcon_api_does_not_overwrite_node_sensor():
    spec, _ = _run_with_iar(
        "  deployNodeSensor: true\n"
        "  falconNodeSensor:\n"
        "    falcon_api:\n"
        "      client_id: node-client\n"
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    falcon_api:\n"
        "      client_id: iar-client\n"
    )
    assert spec["falconClusterGuard"]["falcon_api"]["client_id"] == "node-client"


def test_image_analyzer_image_skipped_with_warning():
    spec, warnings = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    image: my-registry/iar:latest\n"
    )
    assert "image" not in spec["falconClusterGuard"]
    assert any("falconImageAnalyzer.image" in w for w in warnings)


def test_image_analyzer_version_skipped_with_warning():
    _, warnings = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    version: 1.2.3\n"
    )
    assert any("version" in w for w in warnings)


def test_image_analyzer_node_affinity_in_image_analyzer():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    nodeAffinity:\n"
        "      requiredDuringSchedulingIgnoredDuringExecution:\n"
        "        nodeSelectorTerms: []\n"
    )
    assert "nodeAffinity" in spec["falconClusterGuard"]["imageAnalyzer"]


def test_image_analyzer_tolerations_in_image_analyzer():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    tolerations:\n"
        "      - key: node-role.kubernetes.io/master\n"
        "        operator: Exists\n"
    )
    tols = spec["falconClusterGuard"]["imageAnalyzer"]["tolerations"]
    assert tols[0]["key"] == "node-role.kubernetes.io/master"


def test_image_analyzer_registry_crowdstrike_translated():
    spec, warnings = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    registry:\n"
        "      type: crowdstrike\n"
    )
    assert spec["falconClusterGuard"]["registry"]["type"] == "crowdstrike"
    assert not any("mapped to" in w for w in warnings)


def test_image_analyzer_registry_ecr_mapped_to_private():
    spec, warnings = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    registry:\n"
        "      type: ecr\n"
    )
    assert spec["falconClusterGuard"]["registry"]["type"] == "private"
    assert any("ecr" in w and "mapped to 'private'" in w for w in warnings)


# migrate_image_analyzer: imageAnalyzerConfig sub-fields

def test_image_analyzer_config_image_pull_policy_promoted_to_fcg():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    imageAnalyzerConfig:\n"
        "      imagePullPolicy: Always\n"
    )
    assert spec["falconClusterGuard"]["imagePullPolicy"] == "Always"
    assert "imagePullPolicy" not in spec["falconClusterGuard"].get("imageAnalyzer", {})


def test_image_analyzer_config_image_pull_secrets_promoted_to_fcg():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    imageAnalyzerConfig:\n"
        "      imagePullSecrets:\n"
        "        - name: iar-secret\n"
    )
    assert spec["falconClusterGuard"]["imagePullSecrets"][0]["name"] == "iar-secret"
    assert "imagePullSecrets" not in spec["falconClusterGuard"].get("imageAnalyzer", {})


def test_image_analyzer_config_service_account():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    imageAnalyzerConfig:\n"
        "      serviceAccount:\n"
        "        annotations:\n"
        "          eks.amazonaws.com/role-arn: arn:aws:iam::123:role/role\n"
    )
    assert "serviceAccount" in spec["falconClusterGuard"]["imageAnalyzer"]


def test_image_analyzer_config_resources():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    imageAnalyzerConfig:\n"
        "      resources:\n"
        "        limits:\n"
        "          cpu: 500m\n"
    )
    assert spec["falconClusterGuard"]["imageAnalyzer"]["resources"]["limits"]["cpu"] == "500m"


def test_image_analyzer_config_azure_config_path():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    imageAnalyzerConfig:\n"
        "      azureConfigPath: /etc/kubernetes/azure.json\n"
    )
    assert spec["falconClusterGuard"]["imageAnalyzer"]["azureConfigPath"] == "/etc/kubernetes/azure.json"


def test_image_analyzer_config_cluster_name():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    imageAnalyzerConfig:\n"
        "      clusterName: my-cluster\n"
    )
    assert spec["falconClusterGuard"]["imageAnalyzer"]["clusterName"] == "my-cluster"


def test_image_analyzer_config_size_limit():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    imageAnalyzerConfig:\n"
        "      sizeLimit: 50Gi\n"
    )
    assert spec["falconClusterGuard"]["imageAnalyzer"]["sizeLimit"] == "50Gi"


def test_image_analyzer_config_exclusions():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    imageAnalyzerConfig:\n"
        "      exclusions:\n"
        "        namespaces:\n"
        "          - kube-system\n"
    )
    assert "kube-system" in spec["falconClusterGuard"]["imageAnalyzer"]["exclusions"]["namespaces"]


def test_image_analyzer_config_debug():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    imageAnalyzerConfig:\n"
        "      debug: true\n"
    )
    assert spec["falconClusterGuard"]["imageAnalyzer"]["debug"] is True


def test_image_analyzer_config_log_verbosity():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    imageAnalyzerConfig:\n"
        "      logVerbosity: debug\n"
    )
    assert spec["falconClusterGuard"]["imageAnalyzer"]["logVerbosity"] == "debug"


def test_image_analyzer_config_iar_agent_service():
    spec, _ = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    imageAnalyzerConfig:\n"
        "      iarAgentService:\n"
        "        port: 8080\n"
    )
    assert "iarAgentService" in spec["falconClusterGuard"]["imageAnalyzer"]


def test_image_analyzer_config_kac_skipped_with_warning():
    _, warnings = _run_with_iar(
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    imageAnalyzerConfig:\n"
        "      kac:\n"
        "        namespace: falcon-kac\n"
    )
    assert any("kac" in w and "no equivalent" in w for w in warnings)


def test_image_analyzer_config_pull_policy_does_not_overwrite_node_sensor():
    spec, _ = _run_with_iar(
        "  deployNodeSensor: true\n"
        "  falconNodeSensor:\n"
        "    node:\n"
        "      imagePullPolicy: IfNotPresent\n"
        "  deployImageAnalyzer: true\n"
        "  falconImageAnalyzer:\n"
        "    imageAnalyzerConfig:\n"
        "      imagePullPolicy: Always\n"
    )
    assert spec["falconClusterGuard"]["imagePullPolicy"] == "IfNotPresent"


# apply_overrides: image_analyzer override wins over migrated enabled

def test_apply_overrides_image_analyzer_overrides_migrated():
    """CLI --image-analyzer false wins over IAR config already migrated."""
    fd_doc = _make_fd_doc("")
    crs = {"FalconImageAnalyzer": _make_cr_spec(
        "installNamespace: ns\nimageAnalyzerConfig:\n  clusterName: my-cluster"
    )}
    result = assemble(fd_doc, crs, "falcon", [])
    # IAR was enabled (inferred from sub-spec), so enabled=True
    assert result["spec"]["falconClusterGuard"]["imageAnalyzer"]["enabled"] is True
    apply_overrides(result, None, None, image_analyzer=False)
    assert result["spec"]["falconClusterGuard"]["imageAnalyzer"]["enabled"] is False


# assemble: standalone FalconImageAnalyzer CR

def test_assemble_standalone_iar_cr_migrated_to_fcg():
    crs = {"FalconImageAnalyzer": _make_cr_spec(
        "installNamespace: falcon-iar\n"
        "imageAnalyzerConfig:\n"
        "  clusterName: my-cluster\n"
        "  sizeLimit: 30Gi\n"
    )}
    warnings = []
    result = assemble(None, crs, "falcon", warnings)
    fcg = result["spec"]["falconClusterGuard"]
    assert fcg["installNamespace"] == "falcon-iar"
    assert fcg["imageAnalyzer"]["clusterName"] == "my-cluster"
    assert fcg["imageAnalyzer"]["sizeLimit"] == "30Gi"
    assert fcg["imageAnalyzer"]["enabled"] is True
    assert result["spec"]["deployImageAnalyzer"] is False


def test_assemble_standalone_iar_deploy_flag_false_disables_enabled():
    """deployImageAnalyzer: false in the FD disables even if IAR sub-spec is present."""
    fd_doc = _make_fd_doc("  deployImageAnalyzer: false\n  falconImageAnalyzer:\n    installNamespace: ns")
    result = assemble(fd_doc, {}, "falcon", [])
    assert result["spec"]["falconClusterGuard"].get("imageAnalyzer", {}).get("enabled") is not True


# ── main() ───────────────────────────────────────────────────────────────────

def _write_cr_file(tmp_path, filename, kind, spec_yaml=""):
    """Helper to write a CR YAML file for main() tests."""
    _yaml_helper = YAML()
    _yaml_helper.preserve_quotes = True
    cr = CommentedMap()
    cr["apiVersion"] = "falcon.crowdstrike.com/v1alpha1"
    cr["kind"] = kind
    cr["metadata"] = CommentedMap([("name", "test")])
    cr["spec"] = yaml.load(spec_yaml) if spec_yaml else CommentedMap()
    p = tmp_path / filename
    with open(p, "w") as f:
        _yaml_helper.dump(cr, f)
    return p


def test_main_writes_output(tmp_path, monkeypatch):
    p = _write_cr_file(tmp_path, "node.yaml", "FalconNodeSensor", "installNamespace: falcon-system")
    out = tmp_path / "out.yaml"
    monkeypatch.setattr(sys, "argv", ["migrate_to_clusterguard.py", str(p), "-o", str(out)])
    monkeypatch.setattr("builtins.input", lambda _: "y")
    main()
    assert out.exists()
    text = out.read_text()
    assert "FalconDeployment" in text
    assert "falconClusterGuard" in text


def test_main_custom_name(tmp_path, monkeypatch):
    p = _write_cr_file(tmp_path, "node.yaml", "FalconNodeSensor")
    out = tmp_path / "out.yaml"
    monkeypatch.setattr(sys, "argv", ["migrate_to_clusterguard.py", str(p), "-o", str(out), "--name", "my-cluster"])
    monkeypatch.setattr("builtins.input", lambda _: "y")
    main()
    assert "my-cluster" in out.read_text()


def test_main_fcg_image_override(tmp_path, monkeypatch):
    p = _write_cr_file(tmp_path, "node.yaml", "FalconNodeSensor", "installNamespace: ns")
    out = tmp_path / "out.yaml"
    monkeypatch.setattr(sys, "argv", [
        "migrate_to_clusterguard.py", str(p), "-o", str(out),
        "--fcg-image-override", "my-registry/fcg:2.0",
    ])
    monkeypatch.setattr("builtins.input", lambda _: "y")
    main()
    doc = yaml.load(out.read_text())
    assert doc["spec"]["falconClusterGuard"]["image"] == "my-registry/fcg:2.0"
    assert doc["spec"]["falconClusterGuard"]["registry"]["type"] == "private"


def test_main_fcg_image_pull_policy(tmp_path, monkeypatch):
    p = _write_cr_file(tmp_path, "node.yaml", "FalconNodeSensor", "installNamespace: ns")
    out = tmp_path / "out.yaml"
    monkeypatch.setattr(sys, "argv", [
        "migrate_to_clusterguard.py", str(p), "-o", str(out),
        "--fcg-image-pull-policy", "Always",
    ])
    monkeypatch.setattr("builtins.input", lambda _: "y")
    main()
    doc = yaml.load(out.read_text())
    assert doc["spec"]["falconClusterGuard"]["imagePullPolicy"] == "Always"


def test_main_fd_only_migration(tmp_path, monkeypatch):
    p = _write_cr_file(tmp_path, "fd.yaml", "FalconDeployment",
                       "deployNodeSensor: true\nfalconNodeSensor:\n  installNamespace: falcon-system")
    out = tmp_path / "out.yaml"
    monkeypatch.setattr(sys, "argv", ["migrate_to_clusterguard.py", str(p), "-o", str(out)])
    monkeypatch.setattr("builtins.input", lambda _: "y")
    main()
    doc = yaml.load(out.read_text())
    assert doc["spec"]["deployClusterGuard"] is True
    assert "falconClusterGuard" in doc["spec"]


# ── integration: all three scenarios ─────────────────────────────────────────

def test_e2e_scenario1_crds_only():
    """Scenario 1: Only standalone CRDs -> assemble into FD -> FCG migration."""
    crs = {
        "FalconNodeSensor": _make_cr_spec(
            "installNamespace: falcon-system\n"
            "falcon_api:\n  client_id: abc\n"
            "node:\n  backend: bpf\n  tolerations:\n    - key: master\n      operator: Exists"
        ),
        "FalconAdmission": _make_cr_spec(
            "admissionConfig:\n  failurePolicy: Ignore\n  servicePort: 8443"
        ),
    }
    warnings = []
    result = assemble(None, crs, "falcon", warnings)
    fcg = result["spec"]["falconClusterGuard"]
    assert result["spec"]["deployClusterGuard"] is True
    assert result["spec"]["deployNodeSensor"] is False
    assert result["spec"]["deployAdmissionController"] is False
    assert fcg["installNamespace"] == "falcon-system"
    assert fcg["falcon_api"]["client_id"] == "abc"
    assert fcg["nodeSensor"]["backend"] == "bpf"
    assert fcg["controller"]["failurePolicy"] == "Ignore"
    assert fcg["controller"]["servicePort"] == 8443


def test_e2e_scenario2_fd_only():
    """Scenario 2: Only FalconDeployment -> FCG migration."""
    fd_doc = _make_fd_doc(
        "  deployNodeSensor: true\n"
        "  falconNodeSensor:\n"
        "    installNamespace: falcon-system\n"
        "    falcon_api:\n"
        "      client_id: from-fd\n"
        "    node:\n"
        "      backend: bpf\n"
        "  deployAdmissionController: true\n"
        "  falconAdmission:\n"
        "    admissionConfig:\n"
        "      failurePolicy: Ignore\n"
    )
    warnings = []
    result = assemble(fd_doc, {}, "falcon", warnings)
    fcg = result["spec"]["falconClusterGuard"]
    assert result["spec"]["deployClusterGuard"] is True
    assert fcg["installNamespace"] == "falcon-system"
    assert fcg["nodeSensor"]["backend"] == "bpf"
    assert fcg["controller"]["failurePolicy"] == "Ignore"


def test_e2e_scenario3_fd_plus_crds_precedence():
    """Scenario 3: FD + CRDs — FD takes precedence where deploy flag is true."""
    fd_doc = _make_fd_doc(
        "  deployNodeSensor: true\n"
        "  falconNodeSensor:\n"
        "    installNamespace: from-fd\n"
        "    node:\n"
        "      backend: bpf\n"
        "  falcon_api:\n"
        "    client_id: fd-api-key\n"
    )
    crs = {
        "FalconNodeSensor": _make_cr_spec("installNamespace: from-standalone\nnode:\n  backend: kernel"),
        "FalconAdmission": _make_cr_spec("admissionConfig:\n  failurePolicy: Ignore"),
        "FalconContainer": _make_cr_spec("injector:\n  listenPort: 4433"),
    }
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    spec = result["spec"]
    fcg = spec["falconClusterGuard"]

    assert fcg["installNamespace"] == "from-fd"
    assert fcg["nodeSensor"]["backend"] == "bpf"
    assert fcg["falcon_api"]["client_id"] == "fd-api-key"
    assert fcg["controller"]["failurePolicy"] == "Ignore"
    assert spec["falconContainerSensor"]["injector"]["listenPort"] == 4433
    assert spec["deployContainerSensor"] is True
    assert any("FalconNodeSensor" in w and "ignored" in w.lower() for w in warnings)


def test_e2e_overrides_applied_last():
    """CLI overrides win over everything."""
    fd_doc = _make_fd_doc(
        "  deployNodeSensor: true\n"
        "  falconNodeSensor:\n"
        "    installNamespace: ns\n"
        "    node:\n"
        "      image: original:1.0\n"
        "      imagePullPolicy: IfNotPresent\n"
    )
    warnings = []
    result = assemble(fd_doc, {}, "falcon", warnings)
    apply_overrides(result, fcg_image_override="override:2.0", fcg_image_pull_policy="Always")
    fcg = result["spec"]["falconClusterGuard"]
    assert fcg["image"] == "override:2.0"
    assert fcg["registry"]["type"] == "private"
    assert fcg["imagePullPolicy"] == "Always"


def test_e2e_no_yaml_anchors():
    """Output must not contain YAML anchors/aliases."""
    import io as _io
    fd_doc = _make_fd_doc("")
    crs = {
        "FalconNodeSensor": _make_cr_spec("installNamespace: ns\nfalcon_api:\n  client_id: abc"),
        "FalconAdmission": _make_cr_spec("admissionConfig:\n  failurePolicy: Ignore"),
    }
    warnings = []
    result = assemble(fd_doc, crs, "falcon", warnings)
    buf = _io.StringIO()
    yaml.dump(result, buf)
    output = buf.getvalue()
    assert "&id" not in output
    assert "*id" not in output


def test_e2e_scenario4_standalone_iar_cr():
    """Scenario 1 variant: FalconImageAnalyzer standalone CR migrates into falconClusterGuard.imageAnalyzer."""
    crs = {
        "FalconNodeSensor": _make_cr_spec(
            "installNamespace: falcon-system\n"
            "falcon_api:\n  client_id: abc\n"
            "node:\n  backend: bpf\n"
        ),
        "FalconAdmission": _make_cr_spec(
            "admissionConfig:\n  failurePolicy: Ignore\n"
        ),
        "FalconImageAnalyzer": _make_cr_spec(
            "imageAnalyzerConfig:\n"
            "  clusterName: my-cluster\n"
            "  sizeLimit: 30Gi\n"
            "  debug: true\n"
        ),
    }
    warnings = []
    result = assemble(None, crs, "falcon", warnings)
    spec = result["spec"]
    fcg = spec["falconClusterGuard"]

    assert spec["deployClusterGuard"] is True
    assert spec["deployNodeSensor"] is False
    assert spec["deployAdmissionController"] is False
    assert spec["deployImageAnalyzer"] is False
    assert fcg["installNamespace"] == "falcon-system"
    assert fcg["nodeSensor"]["backend"] == "bpf"
    assert fcg["controller"]["failurePolicy"] == "Ignore"
    assert fcg["imageAnalyzer"]["enabled"] is True
    assert fcg["imageAnalyzer"]["clusterName"] == "my-cluster"
    assert fcg["imageAnalyzer"]["sizeLimit"] == "30Gi"
    assert fcg["imageAnalyzer"]["debug"] is True
