# -*- mode: Python -*-

load('ext://cert_manager', 'deploy_cert_manager')

def build_image():
    docker_build(
      "gcr.io/k8s-staging-capi-operator/cluster-api-operator",
      ".",
      ignore=[
        ".git",
        ".github",
        "docs",
        "test",
        "scripts",
        "*.md",
        "LICENSE",
        "OWNERS",
        "OWNERS_ALIASES",
        "PROJECT",
        "SECURITY_CONTACTS"
        ]
    )

def deploy():
    k8s_yaml(
        kustomize('./config/default')
    )

build_image()
deploy_cert_manager()
deploy()
