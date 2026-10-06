#!/usr/bin/env python3
"""
migrate_to_clusterguard.py — Migrate Falcon CRD manifests to FalconClusterGuard.

Accepts any mix of YAML files:
  - FalconDeployment (base document, takes precedence)
  - FalconNodeSensor, FalconAdmission, FalconContainer, FalconImageAnalyzer (standalone CRs)

Three scenarios:
  1. Only standalone CRDs → assembles into a FalconDeployment, then migrates to FCG
  2. Only a FalconDeployment → migrates to FCG
  3. FalconDeployment + standalone CRDs → FD takes precedence, CRDs fill gaps, then FCG migration

FalconDeployment always takes precedence:
  - A standalone CRD is only merged when the FD's corresponding deploy* flag is NOT true
  - FD's shared fields (falcon_api, falconSecret) win over standalone CRD values

Field mapping notes:
  falconNodeSensor:
    installNamespace            -> falconClusterGuard.installNamespace
    falcon_api                  -> falconClusterGuard.falcon_api  (skipped if already set by admission)
    falcon                      -> falconClusterGuard.falcon      (skipped if already set by admission)
    falconSecret                -> falconClusterGuard.falconSecret (skipped if already set by admission)
    node.tolerations            -> falconClusterGuard.nodeSensor.tolerations
    node.nodeAffinity           -> falconClusterGuard.nodeSensor.nodeAffinity
    node.imagePullPolicy        -> falconClusterGuard.imagePullPolicy
    node.image                  -> falconClusterGuard.image
    node.imagePullSecrets       -> falconClusterGuard.imagePullSecrets
    node.updateStrategy         -> falconClusterGuard.nodeSensor.updateStrategy
    node.terminationGracePeriod -> falconClusterGuard.nodeSensor.terminationGracePeriod
    node.serviceAccount         -> falconClusterGuard.nodeSensor.serviceAccount
    node.disableCleanup         -> falconClusterGuard.nodeSensor.disableCleanup
    node.resources              -> falconClusterGuard.nodeSensor.resources
    node.backend                -> falconClusterGuard.nodeSensor.backend
    node.gke                    -> falconClusterGuard.nodeSensor.gke
    node.priorityClass          -> falconClusterGuard.nodeSensor.priorityClass
    node.advanced               -> falconClusterGuard.nodeSensor.advanced
    node.clusterName            -> falconClusterGuard.nodeSensor.clusterName
    node.version                -> (no equivalent, skipped)
    internal                    -> (no equivalent, skipped with warning)

  deployNodeSensor:
    false (or absent)           -> falconClusterGuard.nodeSensor.enabled = false
    true                        -> falconClusterGuard.nodeSensor.enabled is left unset (defaults to true)

  falconAdmission:
    installNamespace            -> falconClusterGuard.installNamespace (skipped if already set)
    falcon_api                  -> falconClusterGuard.falcon_api       (skipped if already set)
    falcon                      -> falconClusterGuard.falcon           (skipped if already set)
    falconSecret                -> falconClusterGuard.falconSecret     (skipped if already set)
    image                       -> falconClusterGuard.image            (skipped if already set)
    registry                    -> falconClusterGuard.registry         (skipped if already set, type differs — verify manually)
    clusterName                 -> falconClusterGuard.controller.clusterName (if present on FalconAdmission CR spec)
    admissionConfig.serviceAccount         -> falconClusterGuard.controller.serviceAccount
    admissionConfig.servicePort            -> falconClusterGuard.controller.servicePort
    admissionConfig.containerPort          -> falconClusterGuard.controller.containerPort
    admissionConfig.tls                    -> falconClusterGuard.controller.tls
    admissionConfig.failurePolicy          -> falconClusterGuard.controller.failurePolicy
    admissionConfig.disabledNamespaces     -> falconClusterGuard.controller.disabledNamespaces
    admissionConfig.watcherEnabled         -> falconClusterGuard.controller.watcherEnabled
    admissionConfig.snapshotsEnabled       -> falconClusterGuard.controller.snapshotsEnabled
    admissionConfig.snapshotsInterval      -> falconClusterGuard.controller.snapshotsInterval
    admissionConfig.admissionControlEnabled  -> falconClusterGuard.controller.admissionControlEnabled
    admissionConfig.configMapWatcherEnabled  -> falconClusterGuard.controller.configMapWatcherEnabled
    admissionConfig.imagePullPolicy          -> falconClusterGuard.imagePullPolicy              (skipped if already set)
    admissionConfig.imagePullSecrets         -> falconClusterGuard.imagePullSecrets             (skipped if already set)
    admissionConfig.falconImageAnalyzerNamespace -> falconClusterGuard.controller.falconImageAnalyzerNamespace
    admissionConfig.replicas                 -> falconClusterGuard.controller.replicas
    admissionConfig.nodeAffinity             -> falconClusterGuard.controller.nodeAffinity
    admissionConfig.tolerations              -> falconClusterGuard.controller.tolerations
    admissionConfig.updateStrategy           -> falconClusterGuard.controller.updateStrategy
    admissionConfig.resources                -> falconClusterGuard.controller.resources
    admissionConfig.resourcesClient          -> falconClusterGuard.controller.resourcesClient
    admissionConfig.resourcesClientNoWebhook -> falconClusterGuard.controller.resourcesClientNoWebhook
    admissionConfig.resourcesWatcher         -> falconClusterGuard.controller.resourcesWatcher
    admissionConfig.deployWatcher            -> (no equivalent, skipped)
    version                                  -> (no equivalent, skipped)
    resourcequota                            -> (no equivalent, skipped with warning)

Usage:
  python3 hack/migration/migrate_to_clusterguard.py node-sensor.yaml admission.yaml -o falcon-deployment.yaml
  python3 hack/migration/migrate_to_clusterguard.py falcon-deployment.yaml -o migrated.yaml
  python3 hack/migration/migrate_to_clusterguard.py fd.yaml node.yaml container.yaml -o migrated.yaml
  python3 hack/migration/migrate_to_clusterguard.py fd.yaml --fcg-image-override my-registry/fcg:1.0 -o out.yaml
  python3 hack/migration/migrate_to_clusterguard.py fd.yaml --fcg-image-pull-policy Always -o out.yaml
  python3 hack/migration/migrate_to_clusterguard.py --interactive
  python3 hack/migration/migrate_to_clusterguard.py -i
"""

import argparse
import copy
import glob
import os
import subprocess
import sys

from ruamel.yaml import YAML
from ruamel.yaml.comments import CommentedMap, CommentedSeq


# ---------------------------------------------------------------------------
# Core migration logic
# ---------------------------------------------------------------------------

# kind -> (FalconDeployment sub-spec key, deploy flag)
_KINDS = {
    "FalconNodeSensor":    ("falconNodeSensor",       "deployNodeSensor"),
    "FalconAdmission":     ("falconAdmission",        "deployAdmissionController"),
    "FalconContainer":     ("falconContainerSensor",  "deployContainerSensor"),
    "FalconImageAnalyzer": ("falconImageAnalyzer",    "deployImageAnalyzer"),
}

# Fields promoted to FalconDeployment top level (first-wins among CRDs, FD always wins).
_SHARED_FIELDS = frozenset(("falcon_api", "falconSecret"))

_PULL_POLICIES = ("Always", "IfNotPresent", "Never")


def _parse_bool(value: str) -> bool:
    """Parse 'true'/'false'/'yes'/'no'/'1'/'0' string into bool."""
    if value.lower() in ("true", "1", "yes"):
        return True
    if value.lower() in ("false", "0", "no"):
        return False
    raise ValueError(f"Expected true/false, got: {value!r}")


def _strip_comments(obj) -> None:
    """Recursively remove all YAML comments from a ruamel.yaml document."""
    if isinstance(obj, CommentedMap):
        obj.ca.items.clear()
        obj.ca.comment = None
        obj.ca.end = None
        for v in obj.values():
            _strip_comments(v)
    elif isinstance(obj, CommentedSeq):
        obj.ca.items.clear()
        obj.ca.comment = None
        obj.ca.end = None
        for item in obj:
            _strip_comments(item)


def _copy(src: CommentedMap, src_key: str, dst: CommentedMap, dst_key: str = None, overwrite: bool = True) -> bool:
    """Copy src[src_key] to dst[dst_key]. Returns True if copied."""
    if dst_key is None:
        dst_key = src_key
    val = src.get(src_key)
    if val is None:
        return False
    if not overwrite and dst_key in dst:
        return False
    dst[dst_key] = copy.deepcopy(val)
    return True


# Registry types that map to "private" in FalconClusterGuard.
_PRIVATE_REGISTRY_TYPES = {"acr", "ecr", "gcr", "openshift"}


def _translate_registry(src_registry: CommentedMap, fcg: CommentedMap, warnings: list) -> None:
    """Translate RegistrySpec to FalconClusterGuardRegistrySpec.

    Old types: crowdstrike, acr, ecr, gcr, openshift  (+ tls, acr_name)
    New types: crowdstrike, private                    (+ tls only)
    """
    if "registry" in fcg:
        warnings.append(
            "falconAdmission.registry was not copied because falconClusterGuard.registry is already set"
        )
        return

    dst = CommentedMap()
    old_type = str(src_registry.get("type", ""))

    if old_type == "crowdstrike":
        dst["type"] = "crowdstrike"
    elif old_type in _PRIVATE_REGISTRY_TYPES:
        dst["type"] = "private"
        warnings.append(
            f"falconAdmission.registry.type '{old_type}' was mapped to 'private'; "
            "FalconClusterGuard does not manage cloud-specific registries (ACR/ECR/GCR/OpenShift) — "
            "set spec.image and spec.imagePullSecrets to pull from your registry"
        )
    elif old_type:
        dst["type"] = "private"
        warnings.append(
            f"falconAdmission.registry.type '{old_type}' is unrecognized and was mapped to 'private'"
        )

    if src_registry.get("tls"):
        dst["tls"] = copy.deepcopy(src_registry["tls"])

    if src_registry.get("acr_name"):
        warnings.append(
            "falconAdmission.registry.acr_name has no equivalent in FalconClusterGuard and was skipped"
        )

    fcg["registry"] = dst


def migrate_node_sensor(node_sensor: CommentedMap, fcg: CommentedMap, warnings: list) -> None:
    # Top-level fields promote to FCG spec
    _copy(node_sensor, "installNamespace", fcg, overwrite=False)
    _copy(node_sensor, "falcon_api", fcg, overwrite=False)
    _copy(node_sensor, "falcon", fcg, overwrite=False)
    _copy(node_sensor, "falconSecret", fcg, overwrite=False)

    if node_sensor.get("internal"):
        warnings.append("falconNodeSensor.internal has no equivalent in FalconClusterGuard and was skipped")

    node = node_sensor.get("node")
    if not node:
        return

    ns = fcg.setdefault("nodeSensor", CommentedMap())

    # node.* fields that promote to FCG spec level
    _copy(node, "imagePullPolicy", fcg, overwrite=False)
    _copy(node, "image", fcg, overwrite=False)
    _copy(node, "imagePullSecrets", fcg, overwrite=False)

    # node.* fields that stay in nodeSensor
    for field in (
        "tolerations",
        "nodeAffinity",
        "updateStrategy",
        "terminationGracePeriod",
        "serviceAccount",
        "disableCleanup",
        "resources",
        "backend",
        "gke",
        "priorityClass",
        "advanced",
        "clusterName",
    ):
        _copy(node, field, ns)


def migrate_admission(admission: CommentedMap, fcg: CommentedMap, warnings: list) -> None:
    # Top-level fields that promote to FCG spec (don't overwrite if node sensor already set them)
    for field in ("installNamespace", "falcon_api", "falcon", "falconSecret", "image"):
        _copy(admission, field, fcg, overwrite=False)

    if admission.get("registry"):
        _translate_registry(admission["registry"], fcg, warnings)

    if admission.get("resourcequota"):
        warnings.append("falconAdmission.resourcequota has no equivalent in FalconClusterGuard and was skipped")

    # clusterName lives on FalconAdmissionSpec directly (not in admissionConfig)
    if admission.get("clusterName"):
        ac_dst = fcg.setdefault("controller", CommentedMap())
        ac_dst["clusterName"] = copy.deepcopy(admission["clusterName"])

    ac_src = admission.get("admissionConfig")
    if not ac_src:
        return

    ac_dst = fcg.setdefault("controller", CommentedMap())

    # These admissionConfig fields promote to FCG top-level (they don't exist on
    # FalconClusterGuardAdmissionSpec, only on FalconClusterGuardSpec).
    for field in (
        "imagePullPolicy",
        "imagePullSecrets",
    ):
        _copy(ac_src, field, fcg, overwrite=False)

    # These admissionConfig fields stay in admissionConfig.
    for field in (
        "serviceAccount",
        "servicePort",
        "containerPort",
        "tls",
        "failurePolicy",
        "disabledNamespaces",
        "watcherEnabled",
        "snapshotsEnabled",
        "snapshotsInterval",
        "admissionControlEnabled",
        "configMapWatcherEnabled",
        "falconImageAnalyzerNamespace",
        "replicas",
        "nodeAffinity",
        "tolerations",
        "updateStrategy",
        "resources",
        "resourcesClient",
        "resourcesClientNoWebhook",
        "resourcesWatcher",
    ):
        _copy(ac_src, field, ac_dst)


def load_files(file_paths: list[str], warnings: list[str]) -> tuple[CommentedMap | None, dict[str, CommentedMap]]:
    _yaml = YAML()
    _yaml.preserve_quotes = True
    _yaml.width = 4096

    fd_doc: CommentedMap | None = None
    fd_path: str | None = None
    crs: dict[str, CommentedMap] = {}
    seen_crs: dict[str, str] = {}  # kind -> file path

    for path in file_paths:
        if not os.path.exists(path):
            print(f"error: file not found: {path!r}", file=sys.stderr)
            sys.exit(1)
        with open(path) as f:
            doc = _yaml.load(f)
        if not isinstance(doc, CommentedMap):
            warnings.append(f"{path}: not a YAML mapping, skipped")
            continue

        kind = doc.get("kind", "")

        if kind == "FalconDeployment":
            if fd_doc is not None:
                print(f"error: duplicate FalconDeployment in {path!r} and {fd_path!r}", file=sys.stderr)
                sys.exit(1)
            fd_doc = doc
            fd_path = path
            continue

        if kind not in _KINDS:
            warnings.append(f"{path}: unrecognized kind {kind!r}, skipped")
            continue

        if kind in seen_crs:
            print(f"error: duplicate kind {kind!r} in {path!r} and {seen_crs[kind]!r}", file=sys.stderr)
            sys.exit(1)
        seen_crs[kind] = path
        spec = doc.get("spec")
        crs[kind] = spec if isinstance(spec, CommentedMap) else CommentedMap()

    if fd_doc is None and not crs:
        print("error: no recognized kinds found in input files", file=sys.stderr)
        sys.exit(1)

    return fd_doc, crs


def _promote_shared(src: CommentedMap, fd_spec: CommentedMap, kind: str,
                    source_map: dict[str, str], warnings: list[str]) -> None:
    """Copy shared fields from a CRD spec to FD spec top level (first-wins among CRDs)."""
    for field in _SHARED_FIELDS:
        val = src.get(field)
        if val is None:
            continue
        if field in fd_spec:
            orig = source_map.get(field, "FalconDeployment")
            warnings.append(f"{kind}.{field} was not copied because it was already set from {orig}")
        else:
            fd_spec[field] = copy.deepcopy(val)
            source_map[field] = kind


def _assemble_component(kind: str, cr_spec: CommentedMap, fd_spec: CommentedMap,
                        source_map: dict[str, str], warnings: list[str]) -> None:
    """Merge a single standalone CRD into the FalconDeployment spec."""
    sub_key, deploy_flag = _KINDS[kind]
    _promote_shared(cr_spec, fd_spec, kind, source_map, warnings)
    sub = fd_spec.setdefault(sub_key, CommentedMap())
    for key, val in cr_spec.items():
        if key not in _SHARED_FIELDS:
            sub[key] = copy.deepcopy(val)
    fd_spec[deploy_flag] = True


def assemble(fd_doc: CommentedMap | None, crs: dict[str, CommentedMap],
             name: str, warnings: list[str]) -> CommentedMap:
    """Combine FD (if any) with standalone CRDs, applying precedence. Returns FD document."""
    if fd_doc is None:
        fd_doc = CommentedMap()
        fd_doc["apiVersion"] = "falcon.crowdstrike.com/v1alpha1"
        fd_doc["kind"] = "FalconDeployment"
        fd_doc["metadata"] = CommentedMap([("name", name)])
        fd_doc["spec"] = CommentedMap()

    spec = fd_doc.get("spec")
    if not isinstance(spec, CommentedMap):
        spec = CommentedMap()
        fd_doc["spec"] = spec

    source_map: dict[str, str] = {}
    # Mark FD's own shared fields so CRDs cannot overwrite them
    for field in _SHARED_FIELDS:
        if field in spec:
            source_map[field] = "FalconDeployment"

    # Merge standalone CRDs where deploy flag is NOT true
    for kind in _KINDS:
        if kind not in crs:
            continue
        _, deploy_flag = _KINDS[kind]
        if spec.get(deploy_flag) is True:
            warnings.append(
                f"{kind} manifest was ignored because {deploy_flag} is already true "
                f"in the FalconDeployment"
            )
            continue
        _assemble_component(kind, crs[kind], spec, source_map, warnings)

    # Set deploy flags: false by default.
    # Exception: FalconImageAnalyzer and FalconContainer infer true from sub-spec presence
    # so that an existing config in the input FD is preserved as enabled.
    _infer_from_spec = frozenset({"FalconImageAnalyzer", "FalconContainer"})
    for kind, (sub_key, deploy_flag) in _KINDS.items():
        if deploy_flag not in spec:
            if kind in _infer_from_spec and bool(spec.get(sub_key)):
                spec[deploy_flag] = True
            else:
                spec[deploy_flag] = False

    # Run FCG migration
    migrate(fd_doc, warnings)

    # Bridge: copy shared fields into falconClusterGuard if not already there
    fcg = spec.get("falconClusterGuard")
    if fcg is not None:
        for field in _SHARED_FIELDS:
            if field in spec and field not in fcg:
                fcg[field] = copy.deepcopy(spec[field])

    return fd_doc


def apply_overrides(fd_doc: CommentedMap, fcg_image_override: str | None,
                    fcg_image_pull_policy: str | None,
                    admission_control: bool | None = None,
                    image_analyzer: bool | None = None) -> None:
    """Apply CLI overrides to falconClusterGuard. Called AFTER assemble() and migrate()."""
    spec = fd_doc.get("spec")
    if not isinstance(spec, CommentedMap):
        return
    fcg = spec.get("falconClusterGuard")
    if fcg is None:
        return

    if fcg_image_override is not None:
        fcg["image"] = fcg_image_override
        registry = fcg.get("registry")
        if not isinstance(registry, CommentedMap):
            registry = CommentedMap()
            fcg["registry"] = registry
        registry["type"] = "private"

    if fcg_image_pull_policy is not None:
        fcg["imagePullPolicy"] = fcg_image_pull_policy

    if admission_control is not None:
        ac = fcg.setdefault("controller", CommentedMap())
        ac["admissionControlEnabled"] = admission_control

    if image_analyzer is not None:
        spec["deployImageAnalyzer"] = image_analyzer


def _group_deploy_flags(spec: CommentedMap) -> None:
    """Reposition deployClusterGuard to sit right after the other deploy* fields."""
    _siblings = frozenset({
        "deployNodeSensor", "deployAdmissionController",
        "deployContainerSensor", "deployImageAnalyzer",
    })
    if "deployClusterGuard" in spec:
        del spec["deployClusterGuard"]
    keys = list(spec.keys())
    last_sibling = max((i for i, k in enumerate(keys) if k in _siblings), default=-1)
    insert_at = last_sibling + 1 if last_sibling >= 0 else len(keys)
    spec.insert(insert_at, "deployClusterGuard", True)


def migrate(doc: CommentedMap, warnings: list) -> None:
    kind = doc.get("kind", "")
    if kind != "FalconDeployment":
        warnings.append(f"expected kind FalconDeployment, got {kind!r} — skipping document")
        return

    spec = doc.get("spec")
    if not isinstance(spec, CommentedMap):
        spec = CommentedMap()
        doc["spec"] = spec

    node_sensor_enabled = spec.get("deployNodeSensor") is not False

    spec["deployNodeSensor"] = False
    spec["deployAdmissionController"] = False

    fcg = spec.setdefault("falconClusterGuard", CommentedMap())

    node_sensor = spec.get("falconNodeSensor")
    if node_sensor:
        migrate_node_sensor(node_sensor, fcg, warnings)
        del spec["falconNodeSensor"]

    if not node_sensor_enabled:
        ns = fcg.setdefault("nodeSensor", CommentedMap())
        ns["enabled"] = False

    admission = spec.get("falconAdmission")
    if admission:
        migrate_admission(admission, fcg, warnings)
        del spec["falconAdmission"]

    # If falconImageAnalyzer or falconContainerSensor config exists and was not explicitly
    # disabled, keep the deploy flag enabled. --image-analyzer false can still override.
    for sub_key, deploy_flag in (("falconImageAnalyzer", "deployImageAnalyzer"),
                                  ("falconContainerSensor", "deployContainerSensor")):
        if spec.get(sub_key) and spec.get(deploy_flag) is not False:
            spec[deploy_flag] = True

    _group_deploy_flags(spec)


# ---------------------------------------------------------------------------
# Interactive wizard — style matches migrate.py (Helm version)
# ---------------------------------------------------------------------------

def _prompt(question: str, default: str = "") -> str:
    """Print a question and return stripped input. Returns default on empty."""
    suffix = f" [{default}]" if default else ""
    try:
        answer = input(f"\n{question}{suffix}: ").strip()
    except (EOFError, KeyboardInterrupt):
        print("\nAborted.", file=sys.stderr)
        sys.exit(1)
    return answer if answer else default


def _choose(question: str, choices: list[tuple[str, str]]) -> tuple[str, str]:
    """Present a numbered menu and return the chosen (label, value) tuple."""
    print(f"\n{question}")
    for i, (label, _) in enumerate(choices, 1):
        print(f"  {i}) {label}")
    while True:
        try:
            raw = input("Enter number: ").strip()
        except (EOFError, KeyboardInterrupt):
            print("\nAborted.", file=sys.stderr)
            sys.exit(1)
        if raw.isdigit() and 1 <= int(raw) <= len(choices):
            return choices[int(raw) - 1]
        print(f"  Please enter a number between 1 and {len(choices)}.")


def _discover_yaml_files() -> list[str]:
    return sorted(glob.glob("*.yaml") + glob.glob("*.yml"))


def _peek_kind(path: str) -> str:
    try:
        _yaml = YAML()
        with open(path) as f:
            doc = _yaml.load(f)
        if isinstance(doc, CommentedMap):
            return str(doc.get("kind", ""))
    except Exception:
        pass
    return ""


def _resolve_file_tokens(tokens: list[str], discovered: list[str]) -> list[str]:
    paths: list[str] = []
    for token in tokens:
        if token.isdigit():
            idx = int(token) - 1
            if 0 <= idx < len(discovered):
                paths.append(discovered[idx])
            else:
                print(f"  warning: index {token} out of range, skipped")
        else:
            paths.append(token)
    return paths


def _peek_ac_iar_defaults(file_paths: list[str]) -> tuple[bool, bool, str, bool]:
    """Peek at input files to derive defaults for the wizard prompts.

    Returns (ac_default, iar_default, name_default, name_from_fd).
    Mirrors the Helm migrate.py approach of reading the current state from the source
    values before asking the user to confirm or change it.
    """
    _yaml = YAML()
    ac_default = True   # admission control on by default (matches old kac default)
    iar_default = False
    name_default = "falcon"
    name_from_fd = False

    for path in file_paths:
        try:
            with open(path) as f:
                doc = _yaml.load(f)
            if not isinstance(doc, CommentedMap):
                continue
            kind = doc.get("kind", "")
            spec = doc.get("spec") or CommentedMap()

            if kind == "FalconDeployment":
                # Inherit metadata.name as the default name
                fd_name = (doc.get("metadata") or {}).get("name", "")
                if fd_name:
                    name_default = fd_name
                    name_from_fd = True
                # deployImageAnalyzer flag takes precedence; if not set but config exists, infer true
                if "deployImageAnalyzer" in spec:
                    iar_default = bool(spec["deployImageAnalyzer"])
                elif spec.get("falconImageAnalyzer"):
                    iar_default = True
                # Already-migrated FCG config
                fcg_ac = ((spec.get("falconClusterGuard") or {})
                          .get("controller") or {}).get("admissionControlEnabled")
                if fcg_ac is not None:
                    ac_default = bool(fcg_ac)
                else:
                    # Pre-migration: read from falconAdmission sub-spec
                    fa_ac = ((spec.get("falconAdmission") or {})
                             .get("admissionConfig") or {}).get("admissionControlEnabled")
                    if fa_ac is not None:
                        ac_default = bool(fa_ac)

            elif kind == "FalconAdmission":
                fa_ac = (spec.get("admissionConfig") or {}).get("admissionControlEnabled")
                if fa_ac is not None:
                    ac_default = bool(fa_ac)

            elif kind == "FalconImageAnalyzer":
                iar_default = True

        except Exception:
            pass

    return ac_default, iar_default, name_default, name_from_fd


def _peek_components(file_paths: list[str]) -> dict[str, bool]:
    """Return enabled state of each Falcon component from the input files."""
    _yaml = YAML()
    states: dict[str, bool] = {
        "FalconNodeSensor":    False,
        "FalconAdmission":     False,
        "FalconContainerSensor": False,
        "FalconImageAnalyzer": False,
    }

    for path in file_paths:
        try:
            with open(path) as f:
                doc = _yaml.load(f)
            if not isinstance(doc, CommentedMap):
                continue
            kind = doc.get("kind", "")
            spec = doc.get("spec") or CommentedMap()

            if kind == "FalconDeployment":
                def _enabled(flag: str, sub_key: str) -> bool:
                    if flag in spec:
                        return bool(spec[flag])
                    return bool(spec.get(sub_key))
                states["FalconNodeSensor"]     = _enabled("deployNodeSensor",          "falconNodeSensor")
                states["FalconAdmission"]      = _enabled("deployAdmissionController", "falconAdmission")
                states["FalconContainerSensor"]= _enabled("deployContainerSensor",     "falconContainerSensor")
                states["FalconImageAnalyzer"]  = _enabled("deployImageAnalyzer",       "falconImageAnalyzer")
            elif kind == "FalconNodeSensor":
                states["FalconNodeSensor"] = True
            elif kind == "FalconAdmission":
                states["FalconAdmission"] = True
            elif kind == "FalconContainer":
                states["FalconContainerSensor"] = True
            elif kind == "FalconImageAnalyzer":
                states["FalconImageAnalyzer"] = True
        except Exception:
            pass

    return states


def print_migration_preview(fd_doc: CommentedMap, output_path: str) -> None:
    sep = "─" * 60
    print(f"\n{sep}")
    print("  Migration preview")
    print(sep)

    spec = fd_doc.get("spec") or CommentedMap()
    name = (fd_doc.get("metadata") or {}).get("name", "")

    print(f"\n  Output:  {output_path}  (name: {name})")

    print("\n  Components:")
    flags = [
        ("deployClusterGuard",        "FalconClusterGuard"),
        ("deployNodeSensor",          "FalconNodeSensor"),
        ("deployAdmissionController", "FalconAdmissionController"),
        ("deployContainerSensor",     "FalconContainerSensor"),
        ("deployImageAnalyzer",       "FalconImageAnalyzer"),
    ]
    for flag, label in flags:
        enabled = bool(spec.get(flag))
        mark = "✓" if enabled else "✗"
        state = "enabled" if enabled else "disabled"
        note = "  (replaced by FalconClusterGuard)" if flag in ("deployAdmissionController", "deployNodeSensor") and not enabled else ""
        print(f"    {mark}  {label:<30}  {state}{note}")

    fcg = spec.get("falconClusterGuard") or CommentedMap()
    if fcg:
        print("\n  falconClusterGuard fields to be written:")
        for key in fcg:
            print(f"    •  {key}")

        ac = fcg.get("controller") or CommentedMap()
        ac_enabled = ac.get("admissionControlEnabled")
        if ac_enabled is not None:
            mark = "✓" if ac_enabled else "✗"
            print(f"\n  {mark}  falconClusterGuard.controller.admissionControlEnabled = {str(bool(ac_enabled)).lower()}")

    print(f"\n{sep}")

    try:
        answer = input("\n  Proceed? [Y/n]: ").strip().lower()
    except (EOFError, KeyboardInterrupt):
        print("\nAborted.", file=sys.stderr)
        sys.exit(1)
    if answer and answer not in ("y", "yes"):
        print("Aborted.", file=sys.stderr)
        sys.exit(0)
    print()


def print_next_steps(output_path: str, input_files: list[str]) -> None:
    sep = "─" * 60
    print(f"\n{sep}")
    print("  Next steps")
    print(sep)
    print(f"""
1) Review the migrated manifest:

   diff {input_files[0]} {output_path}

2) Deploy the new operator (if not already done):

   kubectl apply -f https://github.com/crowdstrike/falcon-operator/releases/latest/download/falcon-operator.yaml
   kubectl -n falcon-operator rollout status deploy/falcon-operator-controller-manager --timeout=120s

3) Apply the migrated FalconDeployment:

   kubectl apply -f {output_path}

4) Verify FalconClusterGuard is running:

   kubectl get falconclusterguard
   kubectl get falconclusterguard -o yaml
""")
    print(sep)


def run_wizard(
    cli_admission_control: bool | None = None,
    cli_image_analyzer: bool | None = None,
) -> tuple[list[str], str, str, str | None, str | None, bool | None, bool | None]:
    """
    Interactive migration wizard. Style matches the Helm migrate.py wizard.

    Returns (file_paths, output_path, name, fcg_image_override, fcg_image_pull_policy,
             admission_control, image_analyzer).
    """
    print("=" * 60)
    print("  FalconDeployment → FalconClusterGuard  Migration Wizard")
    print("=" * 60)

    # ------------------------------------------------------------------
    # Step 1: Migration path
    # ------------------------------------------------------------------
    print("\n── Step 1: Migration path ──")

    path_label, path_key = _choose(
        "Which migration path?",
        [
            ("Path A — migrate an existing FalconDeployment", "A"),
            ("Path B — assemble individual CRs (FalconNodeSensor, FalconAdmission, etc.)", "B"),
        ],
    )
    print(f"  ✓ {path_label}")

    # ------------------------------------------------------------------
    # Step 2: Input files
    # ------------------------------------------------------------------
    print("\n── Step 2: Input files ──")

    discovered = _discover_yaml_files()
    recognized = {"FalconDeployment"} | set(_KINDS.keys())

    if discovered:
        print("\n  YAML files found in the current directory:")
        for i, f in enumerate(discovered, 1):
            kind = _peek_kind(f)
            kind_tag = f"  ({kind})" if kind else ""
            marker = " *" if kind in recognized else ""
            print(f"    {i:2}. {f}{kind_tag}{marker}")
        print("         (* recognized Falcon CRD)")

    if path_key == "A":
        raw = _prompt(
            "FalconDeployment file (number or path)",
            default=next((f for f in discovered if _peek_kind(f) == "FalconDeployment"), ""),
        )
    else:
        raw = _prompt("Input files — numbers or paths, space-separated")

    file_paths = _resolve_file_tokens(raw.split(), discovered) if raw.strip() else []
    if not file_paths:
        print("  ERROR: no input files specified", file=sys.stderr)
        sys.exit(1)
    print(f"  ✓ Using: {', '.join(file_paths)}")

    _comp_states = _peek_components(file_paths)
    # Apply CLI overrides to the display
    if cli_admission_control is not None:
        _comp_states["FalconAdmission"] = cli_admission_control
    if cli_image_analyzer is not None:
        _comp_states["FalconImageAnalyzer"] = cli_image_analyzer

    _FCG_DEST = {
        "FalconNodeSensor":     "migrates to falconClusterGuard.nodeSensor",
        "FalconAdmission":      "migrates to falconClusterGuard.controller",
        "FalconContainerSensor":"preserved in falconContainerSensor",
        "FalconImageAnalyzer":  "preserved in falconImageAnalyzer",
    }
    print()
    for comp, enabled in _comp_states.items():
        mark = "✓" if enabled else "✗"
        note = f"  →  {_FCG_DEST[comp]}" if enabled else ""
        override_note = ""
        if comp == "FalconAdmission" and cli_admission_control is not None:
            override_note = f"  (--admission-control {str(cli_admission_control).lower()})"
        if comp == "FalconImageAnalyzer" and cli_image_analyzer is not None:
            override_note = f"  (--image-analyzer {str(cli_image_analyzer).lower()})"
        print(f"    {mark}  {comp:<26}{note}{override_note}")

    # Derive defaults for Steps 3, 5 & 6 from the input files (mirrors Helm migrate.py behaviour)
    _ac_default, _iar_default, _name_default, _name_from_fd = _peek_ac_iar_defaults(file_paths)

    # ------------------------------------------------------------------
    # Step 3: Output
    # ------------------------------------------------------------------
    print("\n── Step 3: Output ──")

    output_path = _prompt("Output file path", default="falcon-deployment.yaml")
    name_question = (
        "FalconDeployment metadata.name (inherited from existing FalconDeployment)"
        if _name_from_fd else
        "FalconDeployment metadata.name"
    )
    name = _prompt(name_question, default=_name_default)
    print(f"  ✓ Output: {output_path}  (name: {name})")

    # ------------------------------------------------------------------
    # Step 4: FCG image
    # ------------------------------------------------------------------
    print("\n── Step 4: FCG image ──")

    img_label, img_key = _choose(
        "Image source for falconClusterGuard:",
        [
            ("CrowdStrike registry (no override needed)", "crowdstrike"),
            ("Override image — local or air-gapped environment", "override"),
        ],
    )

    fcg_image_override: str | None = None
    fcg_image_pull_policy: str | None = None

    if img_key == "override":
        fcg_image_override = _prompt("  Full image reference (e.g. my-registry/fcg:1.0)")
        if not fcg_image_override:
            print("  Empty value — no image override will be applied.")
            fcg_image_override = None
        else:
            _, policy_val = _choose(
                "  Image pull policy:",
                [(p, p) for p in _PULL_POLICIES],
            )
            fcg_image_pull_policy = policy_val
            print(f"  ✓ Image: {fcg_image_override}  pull policy: {fcg_image_pull_policy}")
    else:
        print(f"  ✓ {img_label}")

    # ------------------------------------------------------------------
    # Step 5: Admission Controller
    # ------------------------------------------------------------------
    print("\n── Step 5: Admission Controller ──")

    if cli_admission_control is not None:
        admission_control: bool | None = cli_admission_control
        print(f"  ✓ admissionControlEnabled = {str(admission_control).lower()} (from --admission-control flag)")
    else:
        ac_prompt_default = "true" if _ac_default else "false"
        print(f"  Current: admissionControlEnabled = {ac_prompt_default}")
        ac_raw = _prompt("Enable admission controller? (true/false)", default=ac_prompt_default)
        try:
            admission_control = _parse_bool(ac_raw)
        except ValueError:
            print(f"  Invalid value '{ac_raw}' — keeping current ({ac_prompt_default})", file=sys.stderr)
            admission_control = _ac_default
        print(f"  ✓ admissionControlEnabled = {str(admission_control).lower()}")

    # ------------------------------------------------------------------
    # Step 6: Image Analyzer
    # ------------------------------------------------------------------
    print("\n── Step 6: Image Analyzer ──")

    if cli_image_analyzer is not None:
        image_analyzer: bool | None = cli_image_analyzer
        print(f"  ✓ deployImageAnalyzer = {str(image_analyzer).lower()} (from --image-analyzer flag)")
    else:
        iar_prompt_default = "true" if _iar_default else "false"
        print(f"  Current: deployImageAnalyzer = {iar_prompt_default}")
        iar_raw = _prompt("Deploy Image Analyzer? (true/false)", default=iar_prompt_default)
        try:
            image_analyzer = _parse_bool(iar_raw)
        except ValueError:
            print(f"  Invalid value '{iar_raw}' — keeping current ({iar_prompt_default})", file=sys.stderr)
            image_analyzer = _iar_default
        print(f"  ✓ deployImageAnalyzer = {str(image_analyzer).lower()}")

    print("\n" + "=" * 60)
    return file_paths, output_path, name, fcg_image_override, fcg_image_pull_policy, admission_control, image_analyzer


_CLUSTER_RESOURCES = [
    ("FalconDeployment",    "falcondeployments"),
    ("FalconAdmission",     "falconadmissions"),
    ("FalconNodeSensor",    "falconnodesensors"),
    ("FalconContainer",     "falconcontainers"),
    ("FalconImageAnalyzer", "falconimageanalyzers"),
]

# These must be removed before the new FalconDeployment manifest can be applied.
_MUST_UNINSTALL = {"falconadmissions", "falconnodesensors", "falconimageanalyzers"}


def show_cluster_resources() -> None:
    """Print existing Falcon CRs found in the cluster (all namespaces)."""
    sep = "─" * 60
    print(f"\n{sep}")
    print("  Existing Falcon resources in the cluster")
    print(sep)

    try:
        subprocess.run(["kubectl", "version", "--client"], capture_output=True, check=True)
    except (FileNotFoundError, subprocess.CalledProcessError):
        print("  (kubectl not found or not configured — skipping cluster query)")
        print(sep)
        return

    found_any = False
    must_remove: list[str] = []

    for kind, plural in _CLUSTER_RESOURCES:
        result = subprocess.run(
            ["kubectl", "get", plural, "-A", "--no-headers", "--ignore-not-found"],
            capture_output=True, text=True,
        )
        lines = [ln for ln in result.stdout.splitlines() if ln.strip()]
        if lines:
            found_any = True
            print(f"\n  {kind}:")
            for line in lines:
                print(f"    {line}")
            if plural in _MUST_UNINSTALL:
                must_remove.append(kind)

    if not found_any:
        print("\n  (no existing Falcon resources found)")

    if must_remove:
        print(f"\n  ⚠  The following resources must be uninstalled before applying")
        print(f"     the migrated FalconDeployment manifest:")
        for kind in must_remove:
            print(f"       • {kind}")
        print(f"\n     Delete each CR and wait for the operator to clean up, e.g.:")
        for kind in must_remove:
            print(f"       kubectl delete {kind.lower()}s --all -A")
        print(f"\n     Then verify they are gone before proceeding:")
        for kind in must_remove:
            print(f"       kubectl get {kind.lower()}s -A")

    print(f"\n{sep}\n")


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def main() -> None:
    parser = argparse.ArgumentParser(
        description=(
            "Migrate Falcon CRD manifests to FalconClusterGuard. "
            "Accepts any mix of FalconDeployment and individual CRs "
            "(FalconNodeSensor, FalconAdmission, FalconContainer, FalconImageAnalyzer). "
            "Run without arguments (or with -i/--interactive) for guided interactive mode."
        )
    )
    parser.add_argument(
        "files", nargs="*",
        help="YAML files containing Falcon CRDs (any mix of kinds)",
    )
    parser.add_argument(
        "-i", "--interactive", action="store_true",
        help="Run the interactive migration wizard (default when no files are given)",
    )
    parser.add_argument("--output", "-o", help="Output file path (default: falcon-deployment.yaml)")
    parser.add_argument("--name", default="falcon",
                        help="metadata.name for the FalconDeployment (default: falcon)")
    parser.add_argument("--fcg-image-override",
                        help="Override falconClusterGuard.image and set registry.type to private")
    parser.add_argument("--fcg-image-pull-policy",
                        help="Override falconClusterGuard.imagePullPolicy (Always, IfNotPresent, Never)")
    parser.add_argument("--admission-control", metavar="BOOL",
                        help="Override falconClusterGuard.controller.admissionControlEnabled (true/false)")
    parser.add_argument("--image-analyzer", metavar="BOOL",
                        help="Override spec.deployImageAnalyzer (true/false)")
    args = parser.parse_args()

    # Parse bool flags early so errors surface before any work begins
    admission_control_override: bool | None = None
    image_analyzer_override: bool | None = None
    if args.admission_control is not None:
        try:
            admission_control_override = _parse_bool(args.admission_control)
        except ValueError as e:
            parser.error(f"--admission-control: {e}")
    if args.image_analyzer is not None:
        try:
            image_analyzer_override = _parse_bool(args.image_analyzer)
        except ValueError as e:
            parser.error(f"--image-analyzer: {e}")

    if args.interactive or not args.files:
        (file_paths, output_path, name, fcg_image_override, fcg_image_pull_policy,
         admission_control_override, image_analyzer_override) = run_wizard(
            cli_admission_control=admission_control_override,
            cli_image_analyzer=image_analyzer_override,
        )
    else:
        file_paths = args.files
        output_path = args.output or "falcon-deployment.yaml"
        name = args.name
        fcg_image_override = args.fcg_image_override
        fcg_image_pull_policy = args.fcg_image_pull_policy

    warnings: list[str] = []
    fd_doc, crs = load_files(file_paths, warnings)
    fd = assemble(fd_doc, crs, name, warnings)
    apply_overrides(fd, fcg_image_override, fcg_image_pull_policy,
                    admission_control=admission_control_override,
                    image_analyzer=image_analyzer_override)

    if warnings:
        print("\nMIGRATION WARNINGS — review these before applying:", file=sys.stderr)
        for w in warnings:
            print(f"  ⚠  {w}", file=sys.stderr)
        print(file=sys.stderr)

    print_migration_preview(fd, output_path)

    _strip_comments(fd)
    _yaml = YAML()
    _yaml.preserve_quotes = True
    _yaml.width = 4096
    with open(output_path, "w") as f:
        _yaml.dump(fd, f)

    print(f"\nMigrated manifest written to: {output_path}")

    show_cluster_resources()

    if args.interactive or not args.files:
        print_next_steps(output_path, file_paths)


if __name__ == "__main__":
    main()
